{{- define "synchronizer.transport.configmap" -}}
{{- printf "%s-%s" "declarations" .Values.SERVICE_NAME | lower | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "synchronizer.preinstall.job" -}}
{{- printf "%s-%s" "synchronizer-preinstall-job" .Values.SERVICE_NAME | lower | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "finalyzer.postinstall.job" -}}
{{- printf "%s-%s" "finalyzer-postinstall-job" .Values.SERVICE_NAME | lower | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "to_millicores" -}}
  {{- $value := toString . -}}
  {{- if hasSuffix "m" $value -}}
    {{ trimSuffix "m" $value }}
  {{- else -}}
    {{ mulf $value 1000 }}
  {{- end -}}
{{- end -}}