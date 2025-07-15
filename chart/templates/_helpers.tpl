{{/*
Expand the name of the chart.
*/}}
{{- define "ServiceName" -}}
{{- default .Chart.Name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/* 
Namespace 

Default is set to service name, its done intentionally, 
Release.namespace should be set to service name as well.
We want to deploy helm chart application to non-default namespace, 
this will allow testing before we plug it into TD routes.

*/}}
{{- define "Namespace" -}}
{{- $service := include "ServiceName" . -}}
{{- .Release.namespace | default $service -}}
{{- end }}

{{- define "NetworkEndpointGroup" -}}
{{- $prefix := "k8s" -}}
{{- $cluster := .Values.cluster | default "primary" -}}
{{- $namespace := include "Namespace" . -}}
{{- $name := include "ServiceName" . -}}
{{- printf "%s-%s-%s-%s" $prefix $cluster $namespace $name -}}
{{- end }}

{{- define "Image" -}}
{{- $registry := "us-central1-docker.pkg.dev/sentryio" -}}
{{- $serviceName := include "ServiceName" . -}}
{{- $repository := .Values.imageRepository | default $serviceName -}}
{{- $name := .Values.imageName | default $serviceName -}}
{{- $tag := .Values.imageTag | default "latest" -}}
{{- printf "%s/%s/%s:%s" $registry $repository $name $tag }}
{{- end }}

{{/*
Labels
*/}}
{{- define "Labels.Cogs" }}
app_feature: {{ .context.Values.cogs.feature }}
app_function: {{ .context.Values.cogs.function }}
{{- end }}

{{- define "Labels.Selector" }}
app.kubernetes.io/instance: {{ .context.Release.Name }} 
service: {{ include "ServiceName" .context }}
{{-     if (hasKey . "customSelectorLabels") }}
{{-         range $key, $value := .customSelectorLabels }}
{{ $key }}: {{ $value }}
{{-         end }}
{{-     end }}
{{- end }}

{{- define "Labels.Deployment" -}}
app.kubernetes.io/managed-by: {{ .context.Release.Service }}
{{-     include "Labels.Selector" . }}
{{-     include "Labels.Cogs" . }}
system: {{ include "ServiceName" .context }}
{{-     if (hasKey . "customDeploymentLabels") }}
{{-         range $key, $value := .customDeploymentLabels }}
{{ $key }}: {{ $value }}
{{-         end }}
{{-     end }}
{{- end }}

{{/*
Annotations
*/}}
{{- define "Annotations.Envoy.Limits" }}
{{- $global := .context.Values.envoy }}
{{- $envoy := .component.envoy | default $global }}
envoy.sentry.io/requestsCpu: "{{ $envoy.requests.cpu }}"
envoy.sentry.io/requestsMemory: "{{ $envoy.requests.memory | default $envoy.limits.memory }}"
envoy.sentry.io/limitsMemory: "{{ $envoy.limits.memory }}"
{{- end }}

{{- define "Annotations.Envoy" }}
sidecar.istio.io/inject: "true"
cloud.google.com/includeOutboundCIDRs: "10.0.0.1/32"
envoy.sentry.io/service: {{ include "ServiceName" .context }}
{{- include "Annotations.Envoy.Limits" . }}
{{- /* Envoy stats collection*/}}
ad.datadoghq.com/envoy.check_names: '["envoy"]'
ad.datadoghq.com/envoy.init_configs: '[{}]'
ad.datadoghq.com/envoy.instances: |
    [
        {
            "openmetrics_endpoint": "http://%%host%%:9902/stats/prometheus",
            "min_collection_interval": 30
        }
    ]
{{- end }}

{{- define "Annotations" }}
cluster-autoscaler.kubernetes.io/safe-to-evict: "true"
{{- include "Annotations.Envoy" . }}
{{- end }}

{{/* 
Init containers 
*/}}
{{- define "DogStatsdPortForward" -}}
{{- $default_version := "latest" -}}
{{- $container := .container | default (dict "version" $default_version) -}}
{{- $version := $container.version | default $default_version -}}
{{- $iptables_entrypoint := (cat 
"iptables -t nat -A OUTPUT -m addrtype --src-type LOCAL --dst-type LOCAL -p udp --dport 8126 -j DNAT --to-destination $HOST_IP:8126"
"iptables -t nat -C POSTROUTING -m addrtype --src-type LOCAL --dst-type UNICAST -j MASQUERADE 2>/dev/null >/dev/null || iptables -t nat -A POSTROUTING -m addrtype --src-type LOCAL --dst-type UNICAST -j MASQUERADE"
) -}}
{{- $output := (
    dict
    "image" "us-central1-docker.pkg.dev/sentryio/iptables/image:{{ $version }}"
    "name" "init-port-forward"
    "args" (
        list
        "/bin/sh"
        "-ec"
        $iptables_entrypoint
    )
    "env" (
        list
        (dict "name" "HOST_IP" "valueFrom" (dict "fieldRef" (dict "fieldPath" "status.hostIP")))
    )
    "securityContext" (
        dict "capabilities" (dict "add" (list "NET_ADMIN"))
    )
) }}
{{- $output | toJson }}
{{- end }}
