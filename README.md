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
| logging.level | Log level Debug, (Info), Warn, Error  |
| logging.format | (text), yaml |
| databaseType | Type of backend Database (mysql), redis or yaml |
| users | List of Users |
| users.username | Username of user for login |
| users.password | Password for user, get hash from -generate (see commandline options)  |
| users.hosts | List of host user can login from ip, CIDR, dns |
| users.permissionsset | List of namespace permissions |
| users.permissionsset.namespaces | List of namespaces covered by permission |
| users.permissionsset.permissions.read | Has read permission if from valid host |
| users.permissionsset.permissions.write | Has write permission if from valid host |
| users.permissionsset.permissions.list | Has list permission if from valid host |
| trustedProxies | List of proxy ipes to trust headders from |
| publicReadableNamespaces | List of namespaces that are public readable |
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

Create test namespace
\[Requires write permission\]  
```bash
curl -u test:test http://localhost:8080/v1 -XPOST -d "test"
201 Created
```

Create Delete namespace
\[Requires write permission\]  
```bash
curl -u test:test http://localhost:8080/v1/test -XDELETE
200 OK
```

List namespace
\[Requires list permission\]  
```bash
curl -u test:test http://localhost:8080/v1/
["kvdb","test"]
```

Set key hello with value world to db.  
Supports POST.  
 \[Requires write permission\]  
```bash
curl -u test:test http://localhost:8080/v1/test/hello -XPOST -d "world"
201 Created
```

Set key hello with value world to db using "value".  
Supports POST.  
 \[Requires write permission\]  
```bash
curl -u test:test http://localhost:8080/v1/test/hello -XPOST -d "value=world"
201 Created
```

Set key hello with value world in json format to db.  
Supports POST.  
 \[Requires write permission\]  
```bash
curl -u test:test http://localhost:8080/v1/test/hello -XPOST -d '{"type": "Key", "value": "world"}' -H 'Content-Type: application/json'
201 Created
```

Put file content of world.txt to key hello in db.  
Supports PUT.  
 \[Requires write permission\]  
```bash
curl -u test:test http://localhost:8080/v1/test/hello -T world.txt
201 Created
```
Note, When writing a complex stucture with Base64 encoding or special charachers use PUT or Post with the pure content.  
If data contains value= be sure to use put. Otherwise the application/x-www-form-urlencoded decoding will fail.


Get key hello from test
\[Requires read permission\]  
```bash
curl -u test:test http://localhost:8080/v1/test/hello
{"key":"hello","namespace":"test","value":"world"}
```

List keys in test namespace  
\[Requires list permission\]  
```bash
curl -u test:test http://localhost:8080/v1/test
["hello"]
```

Delete key hello from db.  
\[Requires write permission\]  
```bash
curl -u test:test http://localhost:8080/v1/test/hello -XDELETE
200 OK
```

Generate random 32 character value for key in json format (Only works if key does not Exists)  
Supports both UPDATE and PATCH for json. Only PATCH for www-form-data.  
\[Requires write permission\]  
```bash
curl  -u test:test http://localhost:8080/v1/test/hello -XUPDATE -d '{"type": "generate"}' -H 'Content-Type: application/json'
{"key":"hello","namespace":"test","value":"4wBZ3VhV9ZoxVjkOz87fQFpnoEe0jCCh"}
```
```bash
curl -u test:test http://localhost:8080/v1/test/hello -XPATCH -d "type=generate"
{"key":"hello","namespace":"test","value":"4wBZ3VhV9ZoxVjkOz87fQFpnoEe0jCCh"}
```

Roll data stored in key to random 32 character value in json format (Only works if key Exists)  
Supports both UPDATE and PATCH for json. Only PATCH for www-form-data.  
\[Requires write permission\]  
```bash
curl -u test:test http://localhost:8080/v1/test/hello -XUPDATE -d '{"type": "roll"}' -H 'Content-Type: application/json'
{"key":"hello","namespace":"test","value":"4wBZ3VhV9ZoxVjkOz87fQFpnoEe0jCCh"}
```
```bash
curl -u test:test http://localhost:8080/v1/test/hello -XPATCH -d "type=roll"
{"key":"hello","namespace":"test","value":"4wBZ3VhV9ZoxVjkOz87fQFpnoEe0jCCh"}
```

Health endpoint  
```bash
curl localhost:8080/system/health -u test:test
{"status":"UP","requests":87}
```

##Public access
config option `publicReadableNamespaces:` allows for a list of namespaces you can read from but not write or list publicly

```bash
curl -u test:test http://localhost:8080/v1/public/hello -XPOST -d "world"
201 Created
```

```bash
curl http://localhost:8080/v1/public/hello
{"key":"hello","namespace":"public","value":"world"}
```