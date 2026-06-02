# ABOUT

Учебный проект по курсу "Highload Architect (Otus)"

## Локальный запуск

### 1. PostgreSQL

Требуется Docker. Из каталога `database`:

```bash
make run-db
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

### 2. Backend

```bash
cd backend
go run ./cmd/app
```

Сервер по умолчанию слушает на порту `:3000` (переменная `HTTP_ADDR`).
URL для подключения к БД задается в переменной `DATABASE_URL`

### 3. Проверка API

**curl:**

```bash
# регистрация
curl -s -X POST http://localhost:3000/user/register \
  -H 'Content-Type: application/json' \
  -d '{"first_name":"Иван","second_name":"Иванов","birthdate":"1990-01-15","biography":"Всегда","city":"Москва","password":"secret"}'

# анкета
curl -s http://localhost:3000/user/get/<user_id>

# логин
curl -s -X POST http://localhost:3000/login \
  -H 'Content-Type: application/json' \
  -d '{"id":"<user_id>","password":"secret"}'
```

**Postman:** импортируйте коллекцию [postman/highload_hw1.json](postman/highload_hw1.json). Запустите запросы по порядку: Register → Get user → Login. Переменные `userId` и `token` заполняются автоматически из ответов.

## API (ДЗ 1)

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/user/register` | Регистрация пользователя |
| GET | `/user/get/{id}` | Получение анкеты |
| POST | `/login` | Аутентификация, выдача токена |
