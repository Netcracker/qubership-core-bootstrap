{{/*
Consul auth mode: CONSUL_AUTH_MODE, else M2M_AUTH_MODE, else legacy.
Fails the render on any value other than legacy, hybrid, or k8s.
*/}}
{{- define "coretpl.consulAuthMode" -}}
{{- $mode := .Values.CONSUL_AUTH_MODE | default .Values.M2M_AUTH_MODE | default "legacy" | trim -}}
{{- if not (has $mode (list "legacy" "hybrid" "k8s")) -}}
{{- fail (printf "Unknown Consul auth mode %q from CONSUL_AUTH_MODE or M2M_AUTH_MODE: expected legacy, hybrid, or k8s" $mode) -}}
{{- end -}}
{{- $mode -}}
{{- end -}}

{{/*
Consul namespace for spec.operatorNamespace of ConsulACL, where the ACL operator runs: the second host label of
CONSUL_URL, http://<consul>.<namespace>[.<domain>]:<port>. Fails the render when that label is missing or numeric,
as in an IP address.
*/}}
{{- define "coretpl.consulNamespace" -}}
{{- $url := .Values.CONSUL_URL | default "" | toString -}}
{{- $parts := splitList "." (urlParse $url).hostname -}}
{{- $namespace := "" -}}
{{- if ge (len $parts) 2 -}}
{{- $namespace = index $parts 1 -}}
{{- end -}}
{{- if or (eq $namespace "") (regexMatch "^[0-9]+$" $namespace) -}}
{{- fail (printf "CONSUL_URL %q names no namespace. Set it to http://<consul>.<namespace>:<port>." $url) -}}
{{- end -}}
{{- $namespace -}}
{{- end -}}

{{/*
"true" in the Consul auth modes hybrid and k8s where the cluster serves the ConsulACL CRD, empty otherwise.
Fails the render in k8s without the CRD, since a service there has no other way to get rights in Consul.
*/}}
{{- define "coretpl.consulAclEnabled" -}}
{{- $mode := include "coretpl.consulAuthMode" . -}}
{{- if has $mode (list "hybrid" "k8s") -}}
{{- if .Capabilities.APIVersions.Has "netcracker.com/v1/ConsulACL" -}}
true
{{- else if eq $mode "k8s" -}}
{{- fail "Consul auth mode k8s needs the ConsulACL CRD (netcracker.com/v1) in the cluster: install Consul with its ACL configurator, or set CONSUL_AUTH_MODE to hybrid or legacy" -}}
{{- end -}}
{{- end -}}
{{- end -}}
