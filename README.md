# CW-RailsPathMetrics
AWS CloudWatchのリクエストパス毎の集計を出力するCLIアプリケーション

## Overview

- AWS CloudWatchから指定した期間のログを取得し、リクエストパス毎に以下を出力します
  - リクエスト回数
  - 平均処理時間
  - 最小処理時間
  - 最大処理時間
- Json形式で結果が出力されます

## How to Use

```
$ cwrstats analyze \
  --start "2025-07-01T00:00:00" \
  --end "2025-07-01T23:59:59" \
  --log-group "/aws/rails/production-log" \
  --profile myprofile

[
    {
        "path": "/path1/path2",
        "count": 100,
        "max_time": 2300ms,
        "min_time": 640ms,
        "avg_time": "1000ms"
    },
    {
        "path": "/path1/path3",
        "count": 50,
        "max_time": 2200ms,
        "min_time": 840ms,
        "avg_time": "1200ms"
    }
]
```

### Option

- --start / --end ログ取得開始時刻と終了時刻（JST）必須
--log-group	CloudWatch Logs のロググループ名 必須
--profile	AWS profile 必須
