# Web Based Key Value Store
This tool is a Key Value database usable as a webhook server.  
Originally build as "Secrets" storage for External Secrets in Kubernetes.  

# Download
Docker image can be fetched from [ghcr.io simonstiil/kvdb](https://github.com/SimonStiil/keyvaluedatabase/pkgs/container/kvdb)  
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
| -config=\[value\] | Use an alternate config filename then config.yaml (only write prefix as .yaml will be appended ) |

## Configuration Structure

| Option | Description ( Defaults ) |
| ------ | ----------- |
| debug | Enable debugging output (developer focused) |
| databaseType | Type of backend Database (mysql), redis or yaml |
| users | List of Users |
| users.username | Username of user for login |
| users.password | Password for user, get hash from -generate (see commandline options)  |
| users.permissions | Permissions of user |
| users.permissions.read | Has read permission if from valid host |
| users.permissions.write | Has write permission if from valid host |
| users.permissions.list | Has list permission if from valid host |
| trustedProxies | List of proxy ipes to trust headders from |
| hosts.address | Limit access by host, Least priviliges of host and user are used |
| hosts.permissions | Permissions of host |
| hosts.permissions.read | Has read permission if valid user |
| hosts.permissions.write | Has write permission if valid user |
| prometheus | Prometheus settings |
| prometheus.enabled | Prometheus enabled (true) |
| prometheus.endpoint | Prometheus endpoint (/system/metrics) |
| redis | Redis settings |
| redis.address | Host address of prometheus server with port (127.0.0.1:6379) |
| redis.envVariableName | Environment value to use for redis password (KVDB_REDIS_PASSWORD) |
| mysql | MySQL settings |
| mysql.address | Host address of prometheus server with port (127.0.0.1:3306) |
| mysql.username | Username to connect to mysql (kvdb) |
| mysql.databaseName | database to connecto to (mysql.username) |
| mysql.tableName | Table to use in database (kvdb) |
| mysql.keyName | Column  to use for key (kvdb) |
| mysql.valueName | Column  to use for value (kvdb) |
| mysql.envVariableName | Environment value to use for redis password (KVDB_MYSQL_PASSWORD) |

## Environmental Options

All configuration options can be set using Environment Values use uppercase and replace . with _ and append KVDB_ prefix.  
Example:
| Option | Description |
| ------ | ----------- |
| KVDB_DEBUG | Enable debugging output (developer focused) |
| KVDB_REDIS_ADDRESS | Hostname for a redis database in format 127.0.0.1:6379 |
| KVDB_REDIS_PASSWORD | Password for Redis database backend |

## Usage
Get key hello from db  
\[Requires GET permission\]  
```bash
curl localhost:8080/hello -u test:test
{"key":"hello","value":"world"}
```

Set key hello with value world to db.  
Supports POST.  
 \[Requires write permission\]  
```bash
curl localhost:8080/hello -u test:test -XPOST -d "world"
OK
```

Set key hello with value world to db using "value".  
Supports POST.  
 \[Requires write permission\]  
```bash
curl localhost:8080/hello -u test:test -XPOST -d "value=world"
OK
```

Set key hello with value world in json format to db.  
Supports POST.  
 \[Requires write permission\]  
```bash
curl localhost:8080/hello -u test:test -XPOST -d '{"value": "world"}' -H 'Content-Type: application/json'
OK
```

Put file content of world.txt to key hello in db.  
Supports PUT.  
 \[Requires write permission\]  
```bash
curl localhost:8080/hello -u test:test -T world.txt
OK
```

Note, When writing a complex stucture with Base64 encoding or special charachers use PUT or Post with the pure content.  
If data contains value= be sure to use put. Otherwise the application/x-www-form-urlencoded decoding will fail.

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