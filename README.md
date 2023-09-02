# Web Based Key Value Store
This tool is a Key Value database usable as a webhook server.  
Originally build as "Secrets" storage for External Secrets in Kubernetes.  

# Download
Docker image can be fetched from [dockerhub simonstiil/kbdv](https://hub.docker.com/repository/docker/simonstiil/kbdv)  
Can be build with go build .  
Will also be available as a release in releases in the future

## Configuration
Is done in config.yaml following the structure  
Example can be seen in [example-config.yaml](./example-config.yaml) 

## Command line options
| Option | Description |
| ------ | ----------- |
| -debug | Enable debugging output (developer focused) |
| -generate=\[value\] | Returns base64 encoded and encrypted password for \[value\] |
| -test=\[output\] | Used with -generate=\[value\] to see if a the generated password matches a the password in \[output\] |
| -config=\[value\] | Use an alternate config filename then config.yaml (only yaml format supported) |
| -port=\[value\] | Use a port different from 8080 |

## Environmental Options

| Option | Description |
| ------ | ----------- |
| KVDB_DEBUG | Enable debugging output (developer focused) |
| KVDB_REDIS_HOST | Hostname for a redis database in format 127.0.0.1:6379 |
| KVDB_REDIS_PASSWORD | Password for a redis database |
| KVDB_AUTH_USERNAME | Additional user from ENV with all rigts |
| KVDB_AUTH_PASSWORD | Password for user from ENV with all rigts |

## Usage
Get key hello from db  
\[Requires GET permission\]  
```bash
curl localhost:8080/hello -u test:test
{"key":"hello","value":"world"}
```

Put key hello with value world to db.  
Supports both PUT and POST.  
 \[Requires write permission\]  
```bash
curl localhost:8080/hello -u test:test -XPUT -d "value=world"
OK
```

Put key hello with value world in json format to db.  
Supports both PUT and POST.  
 \[Requires write permission\]  
```bash
curl localhost:8080/hello -u test:test -XPUT -d '{"value": "world"}' -H 'Content-Type: application/json'
OK
```

Delete key hello from db.  
\[Requires write permission\]  
```bash
curl localhost:8080/hello -u test:test -XDELETE
OK
```

List keys in db  
\[Requires list permission\]  
```bash
curl localhost:8080/system/list -u test:test
["counter","hello"]
```

Generate random 32 character value for key in json format (Only works if key does not Exists)  
Supports both UPDATE and PATCH for json. Only PATCH for www-form-data.  
\[Requires write permission\]  
```bash
curl localhost:8080/hello -XUPDATE -d '{"type": "generate"}' -H 'Content-Type: application/json' -u test:test
{"key":"hello","value":"sBMaPqBPILWLagndcEpq8n27EtydU2m7"}
```
```bash
curl localhost:8080/hello -XPATCH -d "type=generate" -u test:test
{"key":"hello","value":"Nnj169wPuONxmn7OIWkjX49ujAom6Z2O"}
```

Roll data stored in key to random 32 character value in json format (Only works if key Exists)  
Supports both UPDATE and PATCH for json. Only PATCH for www-form-data.  
\[Requires write permission\]  
```bash
curl localhost:8080/hello -XUPDATE -d '{"type": "roll"}' -H 'Content-Type: application/json' -u test:test
{"key":"hello","value":"vubU7vLMJWSeh7sQqCGydJSbyjr4DCRd"}
```
```bash
curl localhost:8080/hello -XPATCH -d "type=roll" -u test:test
{"key":"hello","value":"Llq5q9xuocJBVHoG5ufo1CjIgo9i7YT7"}
```

Health endpoint  
```bash
curl localhost:8080/system/health -u test:test
{"status":"UP","requests":87}
```