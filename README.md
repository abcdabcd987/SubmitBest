# SubmitBest

```bash
go get github.com/revel/revel
go get github.com/revel/cmd/revel
go get github.com/jinzhu/gorm
go get github.com/jinzhu/gorm/dialects/postgres
go get github.com/russross/blackfriday
go get golang.org/x/crypto/bcrypt
```

## Judge Server Configuration

```
FILESERVER_ADDR = "http://center_server:10086/"
TASKSERVER_ADDR = "center_server:10010"

ROOT               = "/tmp/sbest/"
```

```bash
mkdir -p /tmp/sbest
git clone ssh://path/to/center/server/root/problems
```
