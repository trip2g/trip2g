---
layout: mesh/performance_report_ru
free: true
lang_redirect: "[[en/codewriting15]]"

# page meta
title: "trip2g · анатомия 15 месяцев"
description: "Что внутри trip2g после 15 месяцев: 177k строк собственного кода, 139k сгенерированного, 190k вендорного."
version: "v1"
date: "2026-05-25"

# urls
home_url: "/"
github_url: "https://github.com/trip2g"

# multiplier / leverage
multiplier: "×54"
leverage: "54"
typed_by_hand: "6.4"
claude_drafted: "121.7"

# own code breakdown (loc in thousands for display, exact for bars)
loc_typescript: "41,000"
loc_typescript_k: "41"
loc_viewtree: "5,000"
loc_viewtree_k: "5"
loc_go_prod: "32,000"
loc_go_prod_k: "32"
loc_go_tests: "81,000"
loc_go_tests_k: "81"
loc_sql_queries: "2,600"
loc_sql_migrations: "2,500"
loc_graphql: "3,400"
loc_obssync: "9,700"
loc_own: "177"
loc_own_full: "177,000"

# codegen
loc_codegen: "139"
loc_codegen_full: "139,000"
loc_gqlgen: "~78,000"
loc_sqlc: "~43,000"
loc_moq: "~18,000"

# vendor
loc_vendor: "190"
loc_vendor_full: "190,000"

# total
loc_total: "506"
loc_total_full: "506,000"

# sql detail
sql_reads: "1,476"
sql_writes: "1,159"
sql_migration_count: "120"

# segment bar widths (%)
seg_own_pct: "35.0"
seg_codegen_pct: "27.5"
seg_vendor_pct: "37.6"

# stack bar widths (relative to Go tests = 100%)
bar_ts_pct: "50.6%"
bar_go_tests_pct: "100%"
bar_go_prod_pct: "39.5%"
bar_sql_q_pct: "3.2%"
bar_sql_m_pct: "3.1%"

# codegen bar widths (relative to gqlgen = 100%)
bar_gqlgen_pct: "100%"
bar_sqlc_pct: "55.1%"
bar_moq_pct: "23.1%"

# cadence
commits: "1,615"
months: "15"
avg_commits_month: "108"
commits_per_day: "3.6"

# go test ratio
test_ratio: "2.53"
go_total_k: "113"

# waffle
waffle_prod_cells: "57"
waffle_test_cells: "143"

# founder
founder: "Алексей Юрченко"
---
