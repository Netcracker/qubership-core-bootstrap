{{/*
The namespace of the DBaaS installation, for spec.operatorNamespace of the DBaaS Operator CRs.
The DBaaS Operator runs next to the aggregator, so the namespace is read from API_DBAAS_ADDRESS,
http://<aggregator>.<namespace>[.<domain>]:<port>. The render fails when the second label of the
host is missing, empty, or all digits, as in an IP address, since none of those names a namespace.
Call it with the root context:
{{ include "coretpl.dbaasOperatorNamespace" $ }}
*/}}
{{- define "coretpl.dbaasOperatorNamespace" -}}
{{- $host := first (splitList ":" (last (splitList "://" .Values.API_DBAAS_ADDRESS))) -}}
{{- $parts := splitList "." $host -}}
{{- $namespace := "" -}}
{{- if ge (len $parts) 2 -}}
{{- $namespace = index $parts 1 -}}
{{- end -}}
{{- if or (eq $namespace "") (regexMatch "^[0-9]+$" $namespace) -}}
{{- fail (printf "API_DBAAS_ADDRESS %q names no namespace. Set it to http://<aggregator>.<namespace>:<port>." .Values.API_DBAAS_ADDRESS) -}}
{{- end -}}
{{- $namespace -}}
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
