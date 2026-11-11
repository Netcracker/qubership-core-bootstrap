{{/*
The Consul auth mode of Cloud Core: CONSUL_AUTH_MODE, or M2M_AUTH_MODE when it is empty, and legacy when both
are empty. The render fails on any value other than legacy, hybrid, or k8s. Call it with the root context:
{{ include "coretpl.consulAuthMode" $ }}
*/}}
{{- define "coretpl.consulAuthMode" -}}
{{- $mode := .Values.CONSUL_AUTH_MODE | default .Values.M2M_AUTH_MODE | default "legacy" | trim -}}
{{- if not (has $mode (list "legacy" "hybrid" "k8s")) -}}
{{- fail (printf "Unknown Consul auth mode %q from CONSUL_AUTH_MODE or M2M_AUTH_MODE: expected legacy, hybrid, or k8s" $mode) -}}
{{- end -}}
{{- $mode -}}
{{- end -}}

{{/*
The namespace of the Consul installation, for spec.operatorNamespace of the ConsulACL resources.
The ACL operator runs next to Consul, so the namespace is read from CONSUL_URL,
http://<consul>.<namespace>[.<domain>]:<port>. The render fails when the second label of the
host is missing, empty, or all digits, as in an IP address, since none of those names a namespace.
Call it with the root context:
{{ include "coretpl.consulNamespace" $ }}
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
"true" where the chart renders its ConsulACL resource, empty otherwise: in the Consul auth modes hybrid
and k8s, on a cluster that serves the ConsulACL CRD. In k8s a service has no other way to get rights in
Consul, so the render fails there without the CRD. Call it with the root context:
{{- if include "coretpl.consulAclEnabled" $ }}
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
