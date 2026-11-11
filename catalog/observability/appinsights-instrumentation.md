---
name: appinsights-instrumentation
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?((npm|pnpm|yarn|bun)\s+(i|install|add)(\s+((--filter|-F|-w|--workspace|--prefix|-C|--dir|--cwd|--registry)(=[^\s;&|]+|\s+[^\s;&|]+)|-D|--save-dev|-S|--save|-E|--save-exact|-O|--save-optional|-P|--save-prod|-g|--global|-d|--dev|--exact|-W|--ignore-workspace-root-check|--legacy-peer-deps|--no-save|--ignore-scripts|[^\s;&|-][^\s;&|]*))*\s+(@azure/monitor-opentelemetry|applicationinsights)(@[^\s;&|)]*)?($|[\s;&|)])|(pip3?|uv)\s+(install|add|pip\s+install)(\s+((-t|--target|--prefix|-r|--requirement|-c|--constraint|-i|--index-url|--extra-index-url|--root|--src)(=[^\s;&|]+|\s+[^\s;&|]+)|-U|--upgrade|--user|-q|--quiet|--no-deps|--pre|--no-cache-dir|--force-reinstall|-I|--ignore-installed|[^\s;&|-][^\s;&|]*))*\s+azure-monitor-opentelemetry([=<>~!][^\s;&|)]*)?($|[\s;&|)])|dotnet\s+add\b[^;&|\n]*package\s+(Azure\.Monitor\.OpenTelemetry|Microsoft\.ApplicationInsights))
action:
  type: require-skill
  skill: appinsights-instrumentation
upstream:
  repo: github.com/microsoft/azure-skills
  path: skills/appinsights-instrumentation/SKILL.md
  ref: 9aa7f618563713f47481af67b7b9dcaafdc2154a
  content_hash: sha256:43580312e3dc10469be689241e1f52d5d13b65d9fe0134a7b63686e761ce9302
provenance:
  stars: 1636
  license: MIT
  last_push: '2026-11-10'
  fetched: '2026-11-12'
summary: 'Guides instrumenting web apps with Azure Application Insights: SDKs, OpenTelemetry distro and telemetry patterns.'
tags:
- practice
- observability
---
Loads appinsights-instrumentation before an Azure Monitor or Application Insights SDK is installed, so setup follows the reference patterns.
