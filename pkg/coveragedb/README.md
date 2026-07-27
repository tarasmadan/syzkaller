# Prepare coverage aggregation pipeline

Assuming you have the coverage `*.jsonl` files in some bucket:
1. Create BigQuery table.
2. Start data transfers from the bucket to BigQuery table.

Coverage merger job is consuming data from BQ table and store aggregations
in the Spanner DB.

### Create unified BigQuery table

```bash
bq mk \
  --table \
  --description "merged coverage" \
  --time_partitioning_field timestamp \
  --time_partitioning_type DAY \
  --require_partition_filter=true \
  --clustering_fields namespace,file_path,kernel_commit \
  syzkaller:syzbot_coverage.coverage \
  ./pkg/coveragedb/bq-schema.json
```

### Add new data transfer

For each manager/namespace, we set up a Data Transfer to load data from GCS to the unified table.
The `destination_table_name_template` must point to `coverage`.

Example for public manager `upstream`:
```bash
bq mk \
  --transfer_config \
  --display_name=ci-upstream-bucket-to-syzbot_coverage \
  --params='{"destination_table_name_template":"coverage",
  "data_path_template": "gs://$COVERAGE_STREAM_BUCKET/ci-upstream/*.jsonl",
  "allow_jagged_rows": false,
  "allow_quoted_newlines": false,
  "delete_source_files": true,
  "encoding": "UTF8",
  "field_delimiter": ",",
  "file_format": "JSON",
  "ignore_unknown_values": false,
  "max_bad_records": "0",
  "parquet_enable_list_inference": false,
  "parquet_enum_as_string": false,
  "preserve_ascii_control_characters": false,
  "skip_leading_rows": "0",
  "use_avro_logical_types": false,
  "write_disposition": "APPEND"
  }' \
  --project_id=syzkaller \
  --data_source=google_cloud_storage \
  --target_dataset=syzbot_coverage
```

### List BigQuery data transfers
```bash
bq ls \
  --transfer_config \
  --transfer_location=us-central1 \
  --project_id=syzkaller
```