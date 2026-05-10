# file-uploads

Demonstrates multipart upload using the `filesystem` storage backend
together with `sqlite` for metadata.

## Run it

```sh
wave serve examples/apps/file-uploads/server.yaml --port 8106
```

## Try it

```sh
echo "hello wave" > /tmp/hello.txt

# Step 1: write the blob to data/uploads/.
curl -s -F "file=@/tmp/hello.txt" localhost:8106/upload-blob

# Step 2: record the metadata.
curl -s -X POST localhost:8106/upload-meta \
  -H 'content-type: application/json' \
  -d '{"filename":"hello.txt","size":11,"content_type":"text/plain"}'

# List
curl -s localhost:8106/files

# Stream the bytes back (uses the $filetype magic marker).
curl -s localhost:8106/files/by-name/hello.txt
```

## What to look at

- `storage.blobs: type: filesystem` — the filesystem backend exposes
  `WRITE`, `READ`, `DELETE`, `GET_FILE`, `GET_VALUE` as template
  functions; see `infra/filesystem/filesystem.go`.
- `response_content_type: $filetype` is the magic marker that swaps
  the JSON template path for a streaming `Content-Disposition`
  download.

## Caveats

- Two-step upload (blob, then metadata) keeps each `storage-access`
  route bound to a single storage backend. A plugin could fuse both
  into one call.
- No authentication, no per-file quotas, no virus scanning — this is
  a pattern demo, not a production uploader.
