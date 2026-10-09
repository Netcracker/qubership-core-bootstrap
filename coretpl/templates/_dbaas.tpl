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
