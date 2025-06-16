package getters

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/watch"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/tools/record/util"
	ref "k8s.io/client-go/tools/reference"
	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/klog/v2"
	"k8s.io/utils/clock"
	"k8s.io/utils/lru"
)

const (
	maxLruCacheEntries                = 4096
	defaultAggregateMaxEvents         = 10
	defaultAggregateIntervalInSeconds = 600
	defaultSpamBurst                  = 25
	defaultSpamQPS                    = 1. / 300.
	maxTriesPerEvent                  = 12
	maxQueuedEvents                   = 1000
)

var defaultSleepDuration = 10 * time.Second

type CorrelatorOptions struct {
	record.CorrelatorOptions
}

type eventBroadcasterImpl struct {
	*watch.Broadcaster
	sleepDuration  time.Duration
	options        CorrelatorOptions
	cancelationCtx context.Context
	cancel         func()
}

func (e *eventBroadcasterImpl) NewRecorder(scheme *runtime.Scheme, source v1.EventSource) EventRecorderLogger {
	return &recorderImplLogger{recorderImpl: &recorderImpl{scheme, source, e.Broadcaster, clock.RealClock{}}, logger: klog.Background()}
}

func (e *eventBroadcasterImpl) Shutdown() {
	e.Broadcaster.Shutdown()
	e.cancel()
}

func (e *eventBroadcasterImpl) StartEventWatcher(eventHandler func(*v1.Event)) watch.Interface {
	watcher, err := e.Watch()
	if err != nil {
		klog.FromContext(e.cancelationCtx).Error(err, "Unable start event watcher (will not retry!)")
	}
	go func() {
		defer utilruntime.HandleCrash()
		for {
			select {
			case <-e.cancelationCtx.Done():
				watcher.Stop()
				return
			case watchEvent := <-watcher.ResultChan():
				event, ok := watchEvent.Object.(*v1.Event)
				if !ok {
					continue
				}
				eventHandler(event)
			}
		}
	}()
	return watcher
}

func (e *eventBroadcasterImpl) StartLogging(logf func(format string, args ...interface{})) watch.Interface {
	return e.StartEventWatcher(
		func(e *v1.Event) {
			logf("Event(%#v): type: '%v' reason: '%v' %v", e.InvolvedObject, e.Type, e.Reason, e.Message)
		})
}

type EventFilterFunc func(event *v1.Event) bool
type EventAggregatorMessageFunc func(event *v1.Event) string
type EventAggregatorKeyFunc func(event *v1.Event) (aggregateKey string, localKey string)
type EventAggregator struct {
	sync.RWMutex
	cache                *lru.Cache
	keyFunc              EventAggregatorKeyFunc
	messageFunc          EventAggregatorMessageFunc
	maxEvents            uint
	maxIntervalInSeconds uint
	clock                clock.PassiveClock
}

type eventLogger struct {
	sync.RWMutex
	cache *lru.Cache
	clock clock.PassiveClock
}

type EventCorrelator struct {
	filterFunc EventFilterFunc
	aggregator *EventAggregator
	logger     *eventLogger
}

func getEventKey(event *v1.Event) string {
	return strings.Join([]string{
		event.Source.Component,
		event.Source.Host,
		event.InvolvedObject.Kind,
		event.InvolvedObject.Namespace,
		event.InvolvedObject.Name,
		event.InvolvedObject.FieldPath,
		string(event.InvolvedObject.UID),
		event.InvolvedObject.APIVersion,
		event.Type,
		event.Reason,
		event.Message,
	},
		"")
}

type eventLog struct {
	count           uint
	firstTimestamp  metav1.Time
	name            string
	resourceVersion string
}

func (e *eventLogger) updateState(event *v1.Event) {
	key := getEventKey(event)
	e.Lock()
	defer e.Unlock()
	e.cache.Add(
		key,
		eventLog{
			count:           uint(event.Count),
			firstTimestamp:  event.FirstTimestamp,
			name:            event.Name,
			resourceVersion: event.ResourceVersion,
		},
	)
}

func (c *EventCorrelator) UpdateState(event *v1.Event) {
	c.logger.updateState(event)
}

func recordEvent(ctx context.Context, sink EventSink, event *v1.Event, patch []byte, updateExistingEvent bool, eventCorrelator *EventCorrelator) bool {
	var newEvent *v1.Event
	var err error
	if updateExistingEvent {
		newEvent, err = sink.Patch(event, patch)
	}
	if !updateExistingEvent || (updateExistingEvent && util.IsKeyNotFoundError(err)) {
		event.ResourceVersion = ""
		newEvent, err = sink.Create(event)
	}
	if err == nil {
		eventCorrelator.UpdateState(newEvent)
		return true
	}
	switch err.(type) {
	case *restclient.RequestConstructionError:
		klog.FromContext(ctx).Error(err, "Unable to construct event (will not retry!)", "event", event)
		return true
	case *errors.StatusError:
		if errors.IsAlreadyExists(err) || errors.HasStatusCause(err, v1.NamespaceTerminatingCause) {
			klog.FromContext(ctx).V(5).Info("Server rejected event (will not retry!)", "event", event, "err", err)
		} else {
			klog.FromContext(ctx).Error(err, "Server rejected event (will not retry!)", "event", event)
		}
		return true
	case *errors.UnexpectedObjectError:
	default:
	}
	klog.FromContext(ctx).Error(err, "Unable to write event (may retry after sleeping)", "event", event)
	return false
}

type aggregateRecord struct {
	localKeys     sets.String
	lastTimestamp metav1.Time
}

func (e *EventAggregator) EventAggregate(newEvent *v1.Event) (*v1.Event, string) {
	now := metav1.NewTime(e.clock.Now())
	var record aggregateRecord
	eventKey := getEventKey(newEvent)
	aggregateKey, localKey := e.keyFunc(newEvent)
	e.Lock()
	defer e.Unlock()
	value, found := e.cache.Get(aggregateKey)
	if found {
		record = value.(aggregateRecord)
	}
	maxInterval := time.Duration(e.maxIntervalInSeconds) * time.Second
	interval := now.Time.Sub(record.lastTimestamp.Time)
	if interval > maxInterval {
		record = aggregateRecord{localKeys: sets.NewString()}
	}
	record.localKeys.Insert(localKey)
	record.lastTimestamp = now
	e.cache.Add(aggregateKey, record)
	if uint(record.localKeys.Len()) < e.maxEvents {
		return newEvent, eventKey
	}
	record.localKeys.PopAny()
	eventCopy := &v1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%v.%x", newEvent.InvolvedObject.Name, now.UnixNano()),
			Namespace: newEvent.Namespace,
		},
		Count:          1,
		FirstTimestamp: now,
		InvolvedObject: newEvent.InvolvedObject,
		LastTimestamp:  now,
		Message:        e.messageFunc(newEvent),
		Type:           newEvent.Type,
		Reason:         newEvent.Reason,
		Source:         newEvent.Source,
	}
	return eventCopy, aggregateKey
}

func (e *eventLogger) lastEventObservationFromCache(key string) eventLog {
	value, ok := e.cache.Get(key)
	if ok {
		observationValue, ok := value.(eventLog)
		if ok {
			return observationValue
		}
	}
	return eventLog{}
}

func (e *eventLogger) eventObserve(newEvent *v1.Event, key string) (*v1.Event, []byte, error) {
	var (
		patch []byte
		err   error
	)
	eventCopy := *newEvent
	event := &eventCopy
	e.Lock()
	defer e.Unlock()
	lastObservation := e.lastEventObservationFromCache(key)
	if lastObservation.count > 0 {
		event.Name = lastObservation.name
		event.ResourceVersion = lastObservation.resourceVersion
		event.FirstTimestamp = lastObservation.firstTimestamp
		event.Count = int32(lastObservation.count) + 1
		eventCopy2 := *event
		eventCopy2.Count = 0
		eventCopy2.LastTimestamp = metav1.NewTime(time.Unix(0, 0))
		eventCopy2.Message = ""
		newData, _ := json.Marshal(event)
		oldData, _ := json.Marshal(eventCopy2)
		patch, err = strategicpatch.CreateTwoWayMergePatch(oldData, newData, event)
	}
	e.cache.Add(
		key,
		eventLog{
			count:           uint(event.Count),
			firstTimestamp:  event.FirstTimestamp,
			name:            event.Name,
			resourceVersion: event.ResourceVersion,
		},
	)
	return event, patch, err
}

type EventCorrelateResult struct {
	Event *v1.Event
	Patch []byte
	Skip  bool
}

func (c *EventCorrelator) EventCorrelate(newEvent *v1.Event) (*EventCorrelateResult, error) {
	if newEvent == nil {
		return nil, fmt.Errorf("event is nil")
	}
	aggregateEvent, ckey := c.aggregator.EventAggregate(newEvent)
	observedEvent, patch, err := c.logger.eventObserve(aggregateEvent, ckey)
	if c.filterFunc(observedEvent) {
		return &EventCorrelateResult{Skip: true}, nil
	}
	return &EventCorrelateResult{Event: observedEvent, Patch: patch}, err
}

func (e *eventBroadcasterImpl) recordToSink(sink EventSink, event *v1.Event, eventCorrelator *EventCorrelator) {
	eventCopy := *event
	event = &eventCopy
	result, err := eventCorrelator.EventCorrelate(event)
	if err != nil {
		utilruntime.HandleError(err)
	}
	if result.Skip {
		return
	}
	tries := 0
	for {
		if recordEvent(e.cancelationCtx, sink, result.Event, result.Patch, result.Event.Count > 1, eventCorrelator) {
			break
		}
		tries++
		if tries >= maxTriesPerEvent {
			klog.FromContext(e.cancelationCtx).Error(nil, "Unable to write event (retry limit exceeded!)", "event", event)
			break
		}
		delay := e.sleepDuration
		if tries == 1 {
			delay = time.Duration(float64(delay) * rand.Float64())
		}
		select {
		case <-e.cancelationCtx.Done():
			klog.FromContext(e.cancelationCtx).Error(nil, "Unable to write event (broadcaster is shut down)", "event", event)
			return
		case <-time.After(delay):
		}
	}
}

func EventAggregatorByReasonFunc(event *v1.Event) (string, string) {
	return strings.Join([]string{
		event.Source.Component,
		event.Source.Host,
		event.InvolvedObject.Kind,
		event.InvolvedObject.Namespace,
		event.InvolvedObject.Name,
		string(event.InvolvedObject.UID),
		event.InvolvedObject.APIVersion,
		event.Type,
		event.Reason,
		event.ReportingController,
		event.ReportingInstance,
	},
		""), event.Message
}

func EventAggregatorByReasonMessageFunc(event *v1.Event) string {
	return "(combined from similar events): " + event.Message
}

func getSpamKey(event *v1.Event) string {
	return strings.Join([]string{
		event.Source.Component,
		event.Source.Host,
		event.InvolvedObject.Kind,
		event.InvolvedObject.Namespace,
		event.InvolvedObject.Name,
		string(event.InvolvedObject.UID),
		event.InvolvedObject.APIVersion,
	},
		"")
}

func populateDefaults(options CorrelatorOptions) CorrelatorOptions {
	if options.LRUCacheSize == 0 {
		options.LRUCacheSize = maxLruCacheEntries
	}
	if options.BurstSize == 0 {
		options.BurstSize = defaultSpamBurst
	}
	if options.QPS == 0 {
		options.QPS = defaultSpamQPS
	}
	if options.KeyFunc == nil {
		options.KeyFunc = EventAggregatorByReasonFunc
	}
	if options.MessageFunc == nil {
		options.MessageFunc = EventAggregatorByReasonMessageFunc
	}
	if options.MaxEvents == 0 {
		options.MaxEvents = defaultAggregateMaxEvents
	}
	if options.MaxIntervalInSeconds == 0 {
		options.MaxIntervalInSeconds = defaultAggregateIntervalInSeconds
	}
	if options.Clock == nil {
		options.Clock = clock.RealClock{}
	}
	if options.SpamKeyFunc == nil {
		options.SpamKeyFunc = getSpamKey
	}
	return options
}

type EventSpamKeyFunc func(event *v1.Event) string
type EventSourceObjectSpamFilter struct {
	sync.RWMutex
	cache       *lru.Cache
	burst       int
	qps         float32
	clock       clock.PassiveClock
	spamKeyFunc EventSpamKeyFunc
}

func NewEventSourceObjectSpamFilter(lruCacheSize, burst int, qps float32, clock clock.PassiveClock, spamKeyFunc EventSpamKeyFunc) *EventSourceObjectSpamFilter {
	return &EventSourceObjectSpamFilter{
		cache:       lru.New(lruCacheSize),
		burst:       burst,
		qps:         qps,
		clock:       clock,
		spamKeyFunc: spamKeyFunc,
	}
}

func NewEventAggregator(lruCacheSize int, keyFunc EventAggregatorKeyFunc, messageFunc EventAggregatorMessageFunc,
	maxEvents int, maxIntervalInSeconds int, clock clock.PassiveClock) *EventAggregator {
	return &EventAggregator{
		cache:                lru.New(lruCacheSize),
		keyFunc:              keyFunc,
		messageFunc:          messageFunc,
		maxEvents:            uint(maxEvents),
		maxIntervalInSeconds: uint(maxIntervalInSeconds),
		clock:                clock,
	}
}

func newEventLogger(lruCacheEntries int, clock clock.PassiveClock) *eventLogger {
	return &eventLogger{cache: lru.New(lruCacheEntries), clock: clock}
}

type spamRecord struct {
	rateLimiter flowcontrol.PassiveRateLimiter
}

func (f *EventSourceObjectSpamFilter) Filter(event *v1.Event) bool {
	var record spamRecord
	eventKey := f.spamKeyFunc(event)
	f.Lock()
	defer f.Unlock()
	value, found := f.cache.Get(eventKey)
	if found {
		record = value.(spamRecord)
	}
	if record.rateLimiter == nil {
		record.rateLimiter = flowcontrol.NewTokenBucketPassiveRateLimiterWithClock(f.qps, f.burst, f.clock)
	}
	filter := !record.rateLimiter.TryAccept()
	f.cache.Add(eventKey, record)
	return filter
}

func NewEventCorrelatorWithOptions(options CorrelatorOptions) *EventCorrelator {
	optionsWithDefaults := populateDefaults(options)
	spamFilter := NewEventSourceObjectSpamFilter(
		optionsWithDefaults.LRUCacheSize,
		optionsWithDefaults.BurstSize,
		optionsWithDefaults.QPS,
		optionsWithDefaults.Clock,
		EventSpamKeyFunc(optionsWithDefaults.SpamKeyFunc))
	return &EventCorrelator{
		filterFunc: spamFilter.Filter,
		aggregator: NewEventAggregator(
			optionsWithDefaults.LRUCacheSize,
			EventAggregatorKeyFunc(optionsWithDefaults.KeyFunc),
			EventAggregatorMessageFunc(optionsWithDefaults.MessageFunc),
			optionsWithDefaults.MaxEvents,
			optionsWithDefaults.MaxIntervalInSeconds,
			optionsWithDefaults.Clock),
		logger: newEventLogger(optionsWithDefaults.LRUCacheSize, optionsWithDefaults.Clock),
	}
}

func (e *eventBroadcasterImpl) StartRecordingToSink(sink EventSink) watch.Interface {
	eventCorrelator := NewEventCorrelatorWithOptions(e.options)
	return e.StartEventWatcher(
		func(event *v1.Event) {
			e.recordToSink(sink, event, eventCorrelator)
		})
}

func (e *eventBroadcasterImpl) StartStructuredLogging(verbosity klog.Level) watch.Interface {
	loggerV := klog.FromContext(e.cancelationCtx).V(int(verbosity))
	return e.StartEventWatcher(
		func(e *v1.Event) {
			loggerV.Info("Event occurred", "object", klog.KRef(e.InvolvedObject.Namespace, e.InvolvedObject.Name), "fieldPath", e.InvolvedObject.FieldPath, "kind", e.InvolvedObject.Kind, "apiVersion", e.InvolvedObject.APIVersion, "type", e.Type, "reason", e.Reason, "message", e.Message)
		})
}

func (e *eventBroadcasterImpl) NewLabeledRecorder(scheme *runtime.Scheme, source v1.EventSource) EventRecorderLogger {
	return &recorderImplLogger{recorderImpl: &recorderImpl{scheme, source, e.Broadcaster, clock.RealClock{}}, logger: klog.Background()}
}

type config struct {
	CorrelatorOptions
	context.Context
	sleepDuration time.Duration
}

type BroadcasterOption func(*config)
type EventSink interface {
	Create(event *v1.Event) (*v1.Event, error)
	Update(event *v1.Event) (*v1.Event, error)
	Patch(oldEvent *v1.Event, data []byte) (*v1.Event, error)
}

type EventBroadcaster interface {
	StartEventWatcher(eventHandler func(*v1.Event)) watch.Interface
	StartRecordingToSink(sink EventSink) watch.Interface
	StartLogging(logf func(format string, args ...interface{})) watch.Interface
	StartStructuredLogging(verbosity klog.Level) watch.Interface
	NewRecorder(scheme *runtime.Scheme, source v1.EventSource) EventRecorderLogger
	NewLabeledRecorder(scheme *runtime.Scheme, source v1.EventSource) EventRecorderLogger
	Shutdown()
}

func NewBroadcaster(opts ...BroadcasterOption) EventBroadcaster {
	c := config{
		sleepDuration: defaultSleepDuration,
	}
	for _, opt := range opts {
		opt(&c)
	}
	eventBroadcaster := &eventBroadcasterImpl{
		Broadcaster:   watch.NewLongQueueBroadcaster(maxQueuedEvents, watch.DropIfChannelFull),
		sleepDuration: c.sleepDuration,
		options:       c.CorrelatorOptions,
	}
	ctx := c.Context
	if ctx == nil {
		ctx = context.Background()
	} else {
		go func() {
			<-ctx.Done()
			eventBroadcaster.Broadcaster.Shutdown()
		}()
	}
	eventBroadcaster.cancelationCtx, eventBroadcaster.cancel = context.WithCancel(ctx)
	return eventBroadcaster
}

type recorderImpl struct {
	scheme *runtime.Scheme
	source v1.EventSource
	*watch.Broadcaster
	clock clock.PassiveClock
}

type recorderImplLogger struct {
	*recorderImpl
	logger klog.Logger
}

func (recorder recorderImplLogger) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
	recorder.generateEvent(recorder.logger, object, nil, annotations, eventtype, reason, fmt.Sprintf(messageFmt, args...))
}

func (recorder recorderImplLogger) Event(object runtime.Object, eventtype, reason, message string) {
	recorder.recorderImpl.generateEvent(recorder.logger, object, nil, nil, eventtype, reason, message)
}

func (recorder recorderImplLogger) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
	recorder.Event(object, eventtype, reason, fmt.Sprintf(messageFmt, args...))
}

func (recorder recorderImplLogger) WithLogger(logger klog.Logger) EventRecorderLogger {
	return recorderImplLogger{recorderImpl: recorder.recorderImpl, logger: logger}
}

func (recorder *recorderImpl) LabeledEventf(object runtime.Object, labels map[string]string, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
	recorder.generateEvent(klog.Background(), object, labels, annotations, eventtype, reason, fmt.Sprintf(messageFmt, args...))
}

type EventRecorder interface {
	record.EventRecorder
	LabeledEventf(object runtime.Object, labels map[string]string, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{})
}

type EventRecorderLogger interface {
	EventRecorder
	WithLogger(logger klog.Logger) EventRecorderLogger
}

var _ EventRecorderLogger = &recorderImplLogger{}

func (recorder recorderImplLogger) LabeledEventf(object runtime.Object, labels map[string]string, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
	recorder.generateEvent(recorder.logger, object, labels, annotations, eventtype, reason, fmt.Sprintf(messageFmt, args...))
}

func (recorder *recorderImpl) generateEvent(logger klog.Logger, object runtime.Object, labels map[string]string, annotations map[string]string, eventtype, reason, message string) {
	ref, err := ref.GetReference(recorder.scheme, object)
	if err != nil {
		logger.Error(err, "Could not construct reference, will not report event", "object", object, "eventType", eventtype, "reason", reason, "message", message)
		return
	}
	if !util.ValidateEventType(eventtype) {
		logger.Error(nil, "Unsupported event type", "eventType", eventtype)
		return
	}
	event := recorder.makeEvent(ref, labels, annotations, eventtype, reason, message)
	event.Source = recorder.source
	event.ReportingInstance = recorder.source.Host
	event.ReportingController = recorder.source.Component
	sent, err := recorder.ActionOrDrop(watch.Added, event)
	if err != nil {
		logger.Error(err, "Unable to record event (will not retry!)")
		return
	}
	if !sent {
		logger.Error(nil, "Unable to record event: too many queued events, dropped event", "event", event)
	}
}

func (recorder *recorderImpl) makeEvent(ref *v1.ObjectReference, labels map[string]string, annotations map[string]string, eventtype, reason, message string) *v1.Event {
	t := metav1.Time{Time: recorder.clock.Now()}
	namespace := ref.Namespace
	if namespace == "" {
		namespace = metav1.NamespaceDefault
	}
	return &v1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%v.%x", ref.Name, t.UnixNano()),
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		InvolvedObject: *ref,
		Reason:         reason,
		Message:        message,
		FirstTimestamp: t,
		LastTimestamp:  t,
		Count:          1,
		Type:           eventtype,
	}
}
