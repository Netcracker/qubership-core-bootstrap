{{/*
The namespace of the DBaaS installation, for spec.operatorNamespace of the DBaaS Operator CRs.
The DBaaS Operator runs next to the aggregator, so the namespace is read from API_DBAAS_ADDRESS,
http://<aggregator>.<namespace>[.<domain>]:<port>. Call it with the root context:
{{ include "coretpl.dbaasOperatorNamespace" $ }}
*/}}
{{- define "coretpl.dbaasOperatorNamespace" -}}
{{- $host := first (splitList ":" (last (splitList "://" .Values.API_DBAAS_ADDRESS))) -}}
{{- $parts := splitList "." $host -}}
{{- if lt (len $parts) 2 -}}
{{- fail (printf "API_DBAAS_ADDRESS %q names no namespace. Set it to http://<aggregator>.<namespace>:<port>." .Values.API_DBAAS_ADDRESS) -}}
{{- end -}}
{{- index $parts 1 -}}
{{- end -}}

{{/*
"true" where the cluster serves the DBaaS Operator API, empty otherwise. Use it to render the
operator CRs and the mounted operator Secret only where the operator can serve them, so the chart
still installs on clusters without it. Call it with the root context:
{{- if include "coretpl.dbaasOperatorEnabled" $ }}
{{- if not (include "coretpl.dbaasOperatorEnabled" $) }}
*/}}
{{- define "coretpl.dbaasOperatorEnabled" -}}
{{- if .Capabilities.APIVersions.Has "dbaas.netcracker.com/v1" -}}true{{- end -}}
{{- end -}}
