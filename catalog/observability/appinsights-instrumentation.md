---
name: appinsights-instrumentation
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n(])\s*(sudo\s+)?((npm\s+(i|install|add)(\s+((--prefix|-C|--workspace|-w|--registry|--tag)(=[^\s;&|]+|\s+[^\s;&|]+)|-D|--save-dev|-S|--save|-E|--save-exact|-O|--save-optional|-P|--save-prod|-g|--global|--legacy-peer-deps|--no-save|--ignore-scripts|-f|--force|[^\s;&|-][^\s;&|]*))*|pnpm\s+(i|install|add)(\s+((--filter|-F|--dir|-C|--registry)(=[^\s;&|]+|\s+[^\s;&|]+)|-D|--save-dev|-P|--save-prod|-O|--save-optional|-E|--save-exact|-g|--global|-w|--workspace-root|--ignore-scripts|[^\s;&|-][^\s;&|]*))*|yarn\s+(global\s+)?add(\s+((--cwd)(=[^\s;&|]+|\s+[^\s;&|]+)|-D|--dev|-P|--peer|-O|--optional|-E|--exact|-T|--tilde|-W|--ignore-workspace-root-check|[^\s;&|-][^\s;&|]*))*|bun\s+(i|install|add)(\s+((--cwd)(=[^\s;&|]+|\s+[^\s;&|]+)|-d|-D|--dev|--development|-O|--optional|-E|--exact|-g|--global|--peer|[^\s;&|-][^\s;&|]*))*)\s+(@azure/monitor-opentelemetry|applicationinsights)(@[^\s;&|)]*)?($|[\s;&|)])|(pip3?|uv)\s+(install|add|pip\s+install)(\s+((-t|--target|--prefix|-r|--requirement|-c|--constraint|-i|--index-url|--extra-index-url|--root|--src)(=[^\s;&|]+|\s+[^\s;&|]+)|-U|--upgrade|--user|-q|--quiet|--no-deps|--pre|--no-cache-dir|--force-reinstall|-I|--ignore-installed|[^\s;&|-][^\s;&|]*))*\s+azure-monitor-opentelemetry([=<>~!][^\s;&|)]*)?($|[\s;&|)])|dotnet\s+add\b[^;&|\n]*package\s+(Azure\.Monitor\.OpenTelemetry|Microsoft\.ApplicationInsights))
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
