# Cairo Object Storage
Object Storage service for managing unstructured file, organizing content as objects rather than files in a hierarchical firectory tree. 

## Architecture Read & Write File
<img width="1170" height="772" alt="cairo-read-write-object" src="https://github.com/user-attachments/assets/050ea552-6576-4c77-b70b-5d9efd5328b9" />

## Demo dashboard web
[watch demo video](https://www.faissalmaulana.dev/demos/cairo-demo.mp4)

## Run in local

**pre-request**
 - Go 1.26
 - Sqlite (pure Go)
 - Docker
 - Redis
 - Node 24

### To run REST API
```
go mod tidy
make air-server
```


## To see Api Documentation
```
make air-server
curl http://localhost:{PORT}/api/v1/documentation
```
### To run e2e test REST API
```
go mod tidy
go test ./test/...
```

## To run dashboard web
```
pnpm install
pnpm run dev
```
