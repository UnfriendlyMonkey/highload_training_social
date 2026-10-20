# ABOUT

Учебный проект по курсу "Highload Architect (Otus)"

## Локальный запуск

### 1. PostgreSQL

Требуется Docker. Из каталога `database`:

```bash
# для подключения одного инстанса

make run-db

# для запуска кластера
make replication-up
```

Параметры по умолчанию (см. [database/Makefile](database/Makefile)):

| Параметр | Значение |
|----------|----------|
| Host | `localhost` |
| Port | `5435` |
| User | `someuser` |
| Password | `somepass` |
| Database | `socialnet` |

Подключение к БД вручную (psql): `make connect`

**Миграции:** при первом создании БД выполняется `database/migrations/init/init.sql`. После запуска одиночной БД или кластера примените все пронумерованные миграции из корня репозитория (требуется `psql`):

```bash
make migrate
```

Команда применяет файлы по порядку к primary на порту 5435 и останавливается при ошибке. Каждая миграция выполняется в отдельной транзакции. Существующие миграции допускают повторный запуск; новые также должны быть повторно применимыми. `make up` запускает миграции автоматически после готовности реплик. `init.sql` выполняется только при создании пустого хранилища PostgreSQL, а не при каждом запуске контейнера.

### 2. Backend

```bash
cd backend
go run ./cmd/app
```

Сервер по умолчанию слушает на порту `:3000` (переменная `HTTP_ADDR`).
URL для подключения к БД задается в переменной `DATABASE_URL`

Эндпойнты изменения друзей и постов требуют заголовок `Authorization: Bearer <token>` с токеном из `/login`. Чтение поста открыто для всех. Изменять и удалять пост может только автор; чужой или отсутствующий пост возвращает `404`.

Тесты backend выполняются командой `cd backend && go test ./...`. 

Тест использует временную таблицу в отдельной сессии и не меняет данные приложения.

### 3. Проверка API

**curl:**

```bash
# регистрация
curl -s -X POST http://localhost:3000/user/register \
  -H 'Content-Type: application/json' \
  -d '{"first_name":"Иван","second_name":"Иванов","birthdate":"1990-01-15","biography":"Всегда","city":"Москва","password":"secret"}'

# анкета
curl -s http://localhost:3000/user/get/<user_id>

# поиск анкет
curl -s 'http://localhost:3000/user/search?first_name=Ива&last_name=Ива'

# логин
curl -s -X POST http://localhost:3000/login \
  -H 'Content-Type: application/json' \
  -d '{"id":"<user_id>","password":"secret"}'
```

**Postman:** импортируйте коллекцию [postman/highload_hw.json](postman/highload_hw.json). Запустите запросы по порядку: Register → Get user → Search users → Login. Переменные `userId` и `token` заполняются автоматически из ответов.

## API

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/user/register` | Регистрация пользователя |
| GET | `/user/get/{id}` | Получение анкеты |
| GET | `/user/search` | Поиск анкет по префиксам имени и фамилии (`first_name`, `last_name`) |
| POST | `/login` | Аутентификация, выдача токена |
| PUT | `/friend/set/{user_id}` | Добавление друга (нужен токен) |
| PUT | `/friend/delete/{user_id}` | Удаление друга (нужен токен) |
| POST | `/post/create` | Создание поста с JSON `{"text":"..."}` (нужен токен) |
| PUT | `/post/update` | Изменение своего поста с JSON `{"id":"...","text":"..."}` (нужен токен) |
| PUT | `/post/delete/{id}` | Удаление своего поста (нужен токен) |
| GET | `/post/get/{id}` | Получение поста |

