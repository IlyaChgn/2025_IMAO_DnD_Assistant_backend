# Backend-репозиторий проекта Encounterium команды IMAO
## Инструкция по применению миграций
Для применения миграций необходимо запустить приложение командой `go run cmd/app/main.go -migrate <Номер версии>`. В качестве значения флага *migrate* укажите номер той версии миграции, которую нужно применить.

Пример:
``` bash 
$ go run cmd/app/main.go -migrate 1
Migration of version 1 has been applied
```

При применении команды со значением флага latest будет применена последняя версия миграции.

Пример:
``` bash 
$ go run cmd/app/main.go -migrate latest
All migrations have been applied
```