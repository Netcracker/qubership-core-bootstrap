package utils

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var (
	logger      = logging.GetLogger("utils")
	RestyClient = newRestyClient()
)

func MustGetEnv(accessor func(string) string, name string) string {
	value := accessor(name)
	if value == "" {
		logger.Panic("Missed mandatory parameter `%s' value", name)
	}
	return value
}

func GetEnvBoolean(accessor func(string) string, name string) bool {
	value := accessor(name)
	return strings.ToLower(value) == "true"
}

func GeneratePassword(size int) string {
	timestamp := time.Now().UnixNano()
	hash := sha256.Sum256([]byte(fmt.Sprintf("%d", timestamp)))
	password := hex.EncodeToString(hash[:])

	if size < len(password) {
		return password[:size]
	}
	return password
}

func newRestyClient() *resty.Client {
	client := resty.New()
	client.SetDisableWarn(true)

	client.SetTimeout(10 * time.Second)

	client.SetRetryCount(3)
	client.SetRetryWaitTime(2 * time.Second)
	client.SetRetryMaxWaitTime(10 * time.Second)

	client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})

	return client
}

func LogError(log logging.Logger, ctx context.Context, format string, args ...any) error {
	s := fmt.Errorf(format, args...)
	log.ErrorC(ctx, s.Error())
	return s
}

func RegisterShutdownHook(hook func(exitCode int)) {
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)    // Ctrl+C
		signal.Notify(sigint, syscall.SIGTERM) // k8s pre-termination notification

		switch <-sigint {
		case syscall.SIGINT:
			logger.Info("SIGINT signal received, starting shutdown")
			hook(130)
		case syscall.SIGTERM:
			logger.Info("SIGTERM signal received, starting shutdown")
			hook(143)
		}
	}()

}
