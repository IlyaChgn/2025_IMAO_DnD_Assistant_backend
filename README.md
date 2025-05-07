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

### Что делать, если миграция поломалась?
1. Зайти внутрь таблицы schema_migrations и поменять флаг *dirty* с **true** на **false**.
2. Откатиться к предыдущей миграции с помощью команды `go run cmd/app/main.go -migrate <Номер предыдущей версии>`. Например, если поломалась миграция под номером 3, нужно будет откатиться ко второй.
3. Исправить ошибки синтаксиса в новой миграции. 
4. Применить миграцию.

## Для любителей PGAdmin

``` bash
docker-compose -f docker compose.yml -f docker-compose.pgadmin.yml up -d

```
