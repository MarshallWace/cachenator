{{/* vim: set filetype=mustache: */}}

{{/*
Expand the name of the chart.
*/}}
{{- define "cachenator.name" -}}
{{- default .Release.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}


{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "cachenator.fullname" -}}
{{- if hasKey .Values "fullnameOverride" -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}


{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "cachenator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}


{{/*
Common labels
*/}}
{{- define "cachenator.labels" -}}
{{ include "cachenator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end -}}
{{- end -}}

{{/*
Generate the name of the cluster by extracting the last
string after "-" in the namespace
e.g. test or research or production
*/}}
{{- define "cachenator.clusterName" -}}
{{ default (.Release.Namespace | regexFind "[^-]+$" ) .Values.clustername }}
{{- end -}}

{{- define "cachenator.computedFQDN" -}}
{{- .Values.hostname -}}.{{ include "cachenator.clusterName" . }}.mwam.local
{{- end -}}

{{- define "cachenator.computedShortname" -}}
{{- .Values.hostname | regexFind "^[^.]+"  -}}
{{- end -}}


{{/*
Generate name of svc account
i.e. svc-infra-tests-t
*/}}
{{- define "cachenator.svcAccountName" -}}
{{- $namespaceTrimmed := .Release.Namespace | regexFind "-.*-[^-]" | trimAll "-" }}
{{- printf "svc-%s" $namespaceTrimmed }}
{{- end -}}

{{/*
Generate name of svc keytab secret
i.e. svc-infra-tests-t-keytab
*/}}
{{- define "cachenator.svcAccountKeytabName" -}}
{{ $svcName := default (include "cachenator.svcAccountName" .) .Values.KerberosADsvcAccount }}
{{- printf "%s" $svcName }}
{{- end -}}


{{/*
Selector labels
*/}}
{{- define "cachenator.selectorLabels" -}}
app: {{ include "cachenator.name" . }}
{{- with .Values.labels }}
{{ toYaml . }}
{{- end -}}
{{- end -}}


{{/*
Create the name of the service account to use
*/}}
{{- define "cachenator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
    {{ default (include "cachenator.fullname" .) .Values.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.serviceAccount.name }}
{{- end -}}
{{- end -}}


{{/*
Check for existence of nested keys in Values/Maps.
Usage:
{{- if (eq "false" (include "hasDeepKey" (dict "mapToCheck" .Values "keyToFind" "some.nested.key"))) }}
*/}}
{{- define "hasDeepKey" -}}
  {{- $mapToCheck := index . "mapToCheck" -}}
  {{- $keyToFind := index . "keyToFind" -}}
  {{- $keySet := (splitList "." $keyToFind) -}}
  {{- $firstKey := first $keySet -}}
  {{- if index $mapToCheck $firstKey -}}
    {{- if eq 1 (len $keySet) -}}
true
    {{- else }}
      {{- include "hasDeepKey" (dict "mapToCheck" (index $mapToCheck $firstKey) "keyToFind" (join "." (rest $keySet))) }}
    {{- end }}
  {{- else }}
false
  {{- end }}
{{- end }}


{{/*
cachenator env vars
*/}}
{{- define "cachenator.env" -}}
{{- with .Values.envVars -}}
{{ toYaml . | nindent 12 }}
{{- end -}}
{{- end -}}