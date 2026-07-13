# Go Music Server

Личный музыкальный сервер на Go для стриминга локальной библиотеки в iOS-приложение.

Сервер сканирует локальную папку с музыкой, хранит каталог в памяти и отдаёт REST API для просмотра библиотеки, стриминга треков и обложек. Работает на **Windows**, **Linux** и **macOS**. Рассчитан на использование одним пользователем в домашней сети или через VPN.

## Как пользоваться (простая инструкция)

Этот раздел — для тех, кто просто хочет слушать свою музыку с iPhone, без погружения в код.

### Что понадобится

- Компьютер с вашей музыкой (Windows, Linux или macOS)
- **iPhone** с установленным приложением Go Music
- Компьютер и телефон в **одной сети** — домашний Wi‑Fi **или** VPN (например, Amnezia)

> **Linux / macOS:** панель управления с QR-кодом рассчитана на Windows. На Linux и macOS используйте консольный сервер — см. раздел [Запуск на Linux и macOS](#запуск-на-linux-и-macos).

### Шаг 1. Соберите программу (один раз)

Если у вас уже есть `bin\GoMusic.exe`, этот шаг можно пропустить.

Откройте PowerShell в папке проекта и выполните:

```powershell
cd D:\Go_music
go build -o bin\GoMusic.exe .\cmd\gui
```

### Шаг 2. Запустите панель управления

Дважды щёлкните `bin\GoMusic.exe` или в PowerShell:

```powershell
.\bin\GoMusic.exe
```

Откроется браузер со страницей **http://127.0.0.1:8099** — это панель управления.

> **Важно:** не закрывайте окно/процесс `GoMusic.exe`, пока слушаете музыку. Если закрыть — сервер остановится.
>
> Если при повторном запуске пишет, что панель уже запущена — значит, она уже работает. Откройте http://127.0.0.1:8099 в браузере.

### Шаг 3. Настройте сервер в панели

| Поле             | Что указать                                                                           |
| ---------------- | ------------------------------------------------------------------------------------- |
| **Папка музыки** | Путь к вашей музыке, например `D:\Music`                                              |
| **Bearer Token** | Придумайте секретный пароль (или оставьте из `config.yaml`) — его же введёте в iPhone |
| **Host**         | Оставьте **`0.0.0.0`** — так сервер будет доступен и по Wi‑Fi, и по VPN               |
| **Port**         | Обычно **`8080`**                                                                     |

Нажмите **Сохранить**, затем **Запустить**.

После запуска вверху появится статус «Работает» и список адресов для подключения.

### Шаг 4. Подключите iPhone

**Способ А — QR-код (удобнее)**

1. В блоке **«Подключение по QR»** выберите нужную сеть:
   - **VPN (AmneziaVPN)** — если телефон подключён к VPN и вы вне дома
   - **Ethernet / Wi‑Fi** — если телефон в той же домашней сети, что и компьютер
2. Отсканируйте QR-код в iOS-приложении — подставятся адрес сервера и token.

**Способ Б — вручную**

В приложении на iPhone укажите:

| Поле в приложении | Значение                                                                                     |
| ----------------- | -------------------------------------------------------------------------------------------- |
| **Сервер**        | Адрес из панели, например `http://10.8.1.1:8080` (VPN) или `http://192.168.0.93:8080` (дома) |
| **Bearer Token**  | Тот же token, что в панели — **без** слова `Bearer`                                          |

Адреса в панели **настоящие** — они берутся из сетевых интерфейсов компьютера. Поле `0.0.0.0:8080` — это только адрес прослушивания на ПК, **в телефон его вводить не нужно**.

### Шаг 5. Слушайте музыку

Откройте приложение на iPhone — должны появиться исполнители, альбомы и треки с вашего компьютера.

---

### Частые вопросы

**Телефон не подключается**

1. Убедитесь, что в панели статус **«Работает»**
2. Выберите **правильный** адрес: VPN-адрес — только через VPN; домашний IP — только в домашней Wi‑Fi
3. Разрешите порт **8080** в файрволе (Windows — см. ниже; Linux — `ufw allow 8080/tcp`)
4. Проверьте, что token в приложении совпадает с token в панели

**Как открыть порт в брандмауэре Windows**

PowerShell **от имени администратора**:

```powershell
New-NetFirewallRule -DisplayName "Go Music Server" -Direction Inbound -Protocol TCP -LocalPort 8080 -Action Allow
```

**Дома и через VPN — разные адреса**

| Где вы                | Какой адрес использовать                              |
| --------------------- | ----------------------------------------------------- |
| Дома, телефон в Wi‑Fi | `http://192.168.x.x:8080` (Ethernet / Wi‑Fi в панели) |
| Через Amnezia VPN     | `http://10.8.x.x:8080` (VPN в панели)                 |

**Нужно ли менять Host на IP компьютера?**

Нет. Host **`0.0.0.0`** оставляйте как есть. Менять нужно только адрес **в приложении на телефоне** (или QR-код).

**Добавил новую музыку на диск**

Сервер подхватит файлы автоматически. Если что-то не появилось — нажмите **Остановить** → **Запустить** в панели (или `POST /api/rescan`).

**Обложки альбомов не появляются**

При включённом `metadata.enabled` обложки загружаются в фоне. Проверьте `metadata_status` в ответе `/api/albums` — пока `pending`, подождите и обновите список. Нужен доступ в интернет к MusicBrainz.

---

## Возможности

- **Панель управления** в браузере — запуск, остановка, настройки, QR для подключения
- Автоопределение адресов подключения (Wi‑Fi, Ethernet, VPN)
- Автоматическое сканирование библиотеки при запуске
- Поддержка форматов: **MP3**, **FLAC**, **M4A**, **WAV**
- Чтение тегов: title, artist, album, year, genre, duration, track number, embedded cover
- In-memory каталог треков (быстрый доступ без перезагрузки при каждом запросе)
- **Обогащение метаданных** — фоновая загрузка описаний, жанров и обложек альбомов из MusicBrainz (SQLite + файлы на диске)
- Автообновление библиотеки через `fsnotify` (добавление, удаление, переименование файлов)
- HTTP Range Requests для перемотки в AVPlayer
- Bearer Token авторизация
- **Поиск** — `GET /api/search` (артисты, альбомы, треки одним запросом)
- **Станции** — `GET /api/stations` (жанры, десятилетия, всё радио)
- Режим «радио» — по всей библиотеке, артисту, станции или похожим трекам
- Структурированное логирование через `zerolog`

## Требования

- **Go 1.25+**
- **Windows**, **Linux** или **macOS**
- Доступ к папке с музыкой
- Для фонового обогащения метаданных — исходящий доступ в интернет (MusicBrainz, Cover Art Archive)

| Платформа | Консольный сервер (`cmd/server`) | Панель с QR (`cmd/gui`)      |
| --------- | -------------------------------- | ---------------------------- |
| Windows   | ✅                               | ✅ (рекомендуется)           |
| Linux     | ✅                               | ⚠️ без автооткрытия браузера |
| macOS     | ✅                               | ⚠️ без автооткрытия браузера |

## Быстрый старт

### 1. Клонирование и сборка

```powershell
cd D:\Go_music
go build -o bin/server.exe ./cmd/server
```

### 2. Настройка

Отредактируйте `config.yaml` в корне проекта:

```yaml
music_path: D:\Music
data_path: data
host: 0.0.0.0
port: 8080
token: change-me-to-a-secure-token
metadata:
  enabled: true
  user_agent: GoMusic/1.0 (https://github.com/temic/go-music)
```

| Параметр              | Описание                                                                               |
| --------------------- | -------------------------------------------------------------------------------------- |
| `music_path`          | Путь к папке с музыкой                                                                 |
| `data_path`           | Папка для SQLite и скачанных обложек (по умолчанию `data`, рядом с исполняемым файлом) |
| `host`                | Адрес прослушивания (`0.0.0.0` — все интерфейсы)                                       |
| `port`                | Порт HTTP-сервера                                                                      |
| `token`               | Секретный токен для Bearer авторизации                                                 |
| `metadata.enabled`    | Включить фоновое обогащение альбомов через MusicBrainz                                 |
| `metadata.user_agent` | User-Agent для запросов к MusicBrainz (обязателен по правилам API)                     |

Папка `music_path` должна существовать до запуска сервера. Папка `data_path` создаётся автоматически.

### 3. Запуск

**Рекомендуется — панель управления:**

```powershell
go build -o bin\GoMusic.exe .\cmd\gui
.\bin\GoMusic.exe
```

Откроется браузер на `http://127.0.0.1:8099`.

В панели можно:

- указать папку музыки, token, host и port;
- **Запустить** / **Остановить** музыкальный сервер;
- увидеть реальные адреса для подключения с телефона (Wi‑Fi, VPN);
- выбрать сеть и отсканировать **QR-код** для iOS-приложения.

Настройки сохраняются в `config.yaml`.

**Поле Host:** оставляйте `0.0.0.0` — сервер слушает все сетевые интерфейсы. Адрес для телефона смотрите в блоке «Подключение» / QR.

**QR-код** содержит JSON:

```json
{ "server": "http://10.8.1.1:8080", "token": "ваш-token" }
```

**Вариант B — только консоль (без панели):**

```powershell
.\bin\server.exe
```

Или с указанием конфига:

```powershell
.\bin\server.exe -config config.yaml
```

При старте сервер:

1. Загружает конфигурацию
2. Создаёт `data_path` (SQLite и обложки)
3. Сканирует библиотеку
4. Запускает фоновый worker обогащения метаданных (если `metadata.enabled`)
5. Запускает filesystem watcher
6. Поднимает HTTP API

Пример лога:

```
INF http server started music_path=D:\Music data_path=D:\Go_music\data addr=0.0.0.0:8080 tracks=128
```

### 4. Проверка

```bash
curl http://localhost:8080/health
```

```bash
curl -H "Authorization: Bearer change-me-to-a-secure-token" http://localhost:8080/api/tracks
```

## Запуск на Linux и macOS

Консольный сервер полностью поддерживается. Панель управления (`cmd/gui`) собирается, но автооткрытие браузера работает только на Windows — на Linux/macOS откройте панель вручную по адресу http://127.0.0.1:8099.

### 1. Установите Go

Убедитесь, что установлен Go 1.25 или новее:

```bash
go version
```

### 2. Сборка

```bash
cd /path/to/Go_music
go build -o bin/server ./cmd/server
```

Опционально — панель управления:

```bash
go build -o bin/gomusic-gui ./cmd/gui
```

### 3. Настройка `config.yaml`

Пример для Linux:

```yaml
music_path: /home/user/Music
data_path: data
host: 0.0.0.0
port: 8080
token: change-me-to-a-secure-token
metadata:
  enabled: true
  user_agent: GoMusic/1.0 (https://github.com/temic/go-music)
```

Пример для macOS:

```yaml
music_path: /Users/you/Music
data_path: data
host: 0.0.0.0
port: 8080
token: change-me-to-a-secure-token
metadata:
  enabled: true
  user_agent: GoMusic/1.0 (https://github.com/temic/go-music)
```

### 4. Запуск

```bash
./bin/server -config config.yaml
```

Сервер слушает `0.0.0.0:8080` на всех сетевых интерфейсах. Узнать IP компьютера:

```bash
# Linux
ip -4 addr show | grep inet

# macOS
ipconfig getifaddr en0
```

В iOS-приложении укажите `http://<IP-компьютера>:8080` и тот же `token`, что в конфиге.

### 5. Файрвол

**Linux (ufw):**

```bash
sudo ufw allow 8080/tcp
```

**macOS:** Системные настройки → Сеть → Файрвол → Параметры → разрешите входящие подключения для `server` (или временно отключите файрвол для проверки).

### 6. Автозапуск (опционально)

Пример unit-файла systemd на Linux (`/etc/systemd/system/gomusic.service`):

```ini
[Unit]
Description=Go Music Server
After=network.target

[Service]
Type=simple
User=music
WorkingDirectory=/opt/gomusic
ExecStart=/opt/gomusic/bin/server -config /opt/gomusic/config.yaml
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now gomusic
```

## Структура музыкальной библиотеки

Структура папок может быть любой. Рекомендуемый вариант:

```
Music/                          # D:\Music на Windows, /home/user/Music на Linux
    Linkin Park/
        Meteora/
            01 - Foreword.mp3
            02 - Don't Stay.mp3
            cover.jpg

    Queen/
        Greatest Hits/
            01.mp3
            02.mp3
```

### Как определяются метаданные

1. **Из тегов файла** (приоритет) — через библиотеку `dhowden/tag`
2. **Fallback без тегов:**
   - `title` — из имени файла (префикс `01 - ` убирается)
   - `artist` / `album` — из структуры папок (`Artist/Album/track.mp3`)

### Обложки

**Треки** — порядок поиска для `GET /api/tracks/{id}/cover`:

1. Встроенная обложка из тегов файла
2. Файл в папке трека: `cover.jpg`, `folder.jpg`, `front.jpg`

**Альбомы** — порядок для `GET /api/albums/{id}/cover`:

1. Скачанная обложка из MusicBrainz (если включено `metadata.enabled`)
2. Встроенная обложка из тегов треков альбома
3. Файл `cover.jpg` / `folder.jpg` / `front.jpg` в папке альбома

## Обогащение метаданных

При `metadata.enabled: true` сервер в фоне обогащает альбомы через [MusicBrainz](https://musicbrainz.org/) и [Cover Art Archive](https://coverartarchive.org/):

1. При индексации трека альбом ставится в очередь (не блокирует сканирование)
2. Фоновый worker запрашивает MusicBrainz по artist + album + year
3. Скачанная обложка сохраняется в `data/covers/{albumId}.jpg`
4. Результаты — в SQLite `data/library.db`

Данные каталога (треки, артисты) по-прежнему в памяти; SQLite используется только для обогащения альбомов.

### Поля альбома из обогащения

| Поле              | Описание                                                                         |
| ----------------- | -------------------------------------------------------------------------------- |
| `cover_url`       | Относительный URL обложки, напр. `/api/albums/{id}/cover` (требует Bearer token) |
| `genres`          | Жанры из MusicBrainz                                                             |
| `description`     | Описание / аннотация альбома                                                     |
| `musicbrainz_id`  | ID релиза в MusicBrainz                                                          |
| `metadata_status` | `pending`, `success`, `failed` или `skipped`                                     |

Пока `metadata_status` равен `pending`, клиент может периодически обновлять список альбомов — обложка и описание появятся после завершения фоновой загрузки.

Чтобы отключить обогащение (офлайн-режим, без интернета):

```yaml
metadata:
  enabled: false
```

## Авторизация

Все эндпоинты `/api/*` требуют заголовок:

```
Authorization: Bearer <token>
```

где `<token>` — значение из `config.yaml`.

Эндпоинт `/health` доступен без авторизации.

При ошибке авторизации:

```json
{
  "error": true,
  "message": "Invalid token"
}
```

HTTP статус: `401 Unauthorized`

## REST API

Базовый URL: `http://<host>:<port>`

### Формат ошибок

Все ошибки API возвращаются в JSON:

```json
{
  "error": true,
  "message": "Track not found"
}
```

---

### `GET /health`

Проверка доступности сервера. Авторизация не требуется.

**Ответ `200 OK`:**

```json
{
  "status": "ok"
}
```

---

### `GET /api/artists`

Список исполнителей с пагинацией.

**Query-параметры:** `page` (по умолчанию `1`), `limit` (по умолчанию `50`, макс. `200`)

**Ответ `200 OK`:**

```json
{
  "items": [
    {
      "artist": "a1b2c3...",
      "name": "Queen",
      "album_count": 3,
      "track_count": 42,
      "duration": 7234.5
    }
  ],
  "total": 1,
  "page": 1,
  "limit": 50,
  "total_pages": 1
}
```

| Поле          | Тип    | Описание                             |
| ------------- | ------ | ------------------------------------ |
| `artist`      | string | Стабильный ID исполнителя            |
| `name`        | string | Имя исполнителя                      |
| `album_count` | int    | Количество альбомов                  |
| `track_count` | int    | Количество треков                    |
| `duration`    | float  | Суммарная длительность (**секунды**) |

---

### `GET /api/albums`

Список альбомов с пагинацией.

**Query-параметры:** `page`, `limit`, `artist_id` (фильтр по ID исполнителя из `/api/artists`)

**Пример — альбомы одного исполнителя:**

```
GET /api/albums?artist_id=a1b2c3...
```

**Ответ `200 OK`:**

```json
{
  "items": [
    {
      "id": "d4e5f6...",
      "title": "Greatest Hits",
      "artist": "Queen",
      "year": 1981,
      "track_count": 17,
      "duration": 4123.8,
      "has_cover": true,
      "cover_url": "/api/albums/d4e5f6.../cover",
      "genres": ["Rock", "Pop Rock"],
      "description": "Greatest hits compilation.",
      "musicbrainz_id": "b1234567-...",
      "metadata_status": "success"
    }
  ],
  "total": 1,
  "page": 1,
  "limit": 50,
  "total_pages": 1
}
```

| Поле (доп.)       | Тип      | Описание                                                     |
| ----------------- | -------- | ------------------------------------------------------------ |
| `cover_url`       | string   | URL обложки (если есть); требует авторизации                 |
| `genres`          | string[] | Жанры из MusicBrainz (если обогащено)                        |
| `description`     | string   | Описание альбома                                             |
| `musicbrainz_id`  | string   | ID в MusicBrainz                                             |
| `metadata_status` | string   | Статус обогащения: `pending`, `success`, `failed`, `skipped` |

Поля обогащения опциональны — при `metadata.enabled: false` возвращаются только базовые поля.

---

### `GET /api/albums/{id}/tracks`

Треки альбома.

**Ответ `200 OK`:** массив объектов `Track`

---

### `GET /api/albums/{id}/cover`

Обложка альбома.

Порядок: скачанная обложка (MusicBrainz) → встроенная в теги → `cover.jpg` в папке.

**Ответ `200 OK`:** бинарное изображение (`image/jpeg` для скачанных обложек)

---

### `GET /api/search`

Единый поиск по библиотеке для экрана поиска в iOS.

**Query-параметры:**

| Параметр | Тип    | По умолчанию | Описание                            |
| -------- | ------ | ------------ | ----------------------------------- |
| `q`      | string | —            | Поисковый запрос                    |
| `limit`  | int    | `20`         | Лимит на каждую группу (макс. `50`) |

Ищет:

- **треки** — по title, artist, album, genre
- **исполнителей** — по имени или если у них есть подходящие треки
- **альбомы** — по названию, исполнителю или трекам внутри альбома

**Пример:**

```
GET /api/search?q=leon&limit=10
```

**Ответ `200 OK`:**

```json
{
  "query": "leon",
  "artists": [
    {
      "artist": "a1b2c3...",
      "name": "Kings of Leon",
      "album_count": 6,
      "track_count": 42,
      "duration": 7234.5
    }
  ],
  "albums": [
    {
      "id": "d4e5f6...",
      "title": "Only by the Night",
      "artist": "Kings of Leon",
      "track_count": 11,
      "duration": 2123.8,
      "has_cover": true
    }
  ],
  "tracks": [
    {
      "id": "9f86d081...",
      "title": "Sex on Fire",
      "artist": "Kings of Leon",
      "album": "Only by the Night",
      "duration": 203.4,
      "format": "flac"
    }
  ]
}
```

---

### `GET /api/stations`

Список «станций» — подборок для таба **Станции** в iOS.

Станции строятся автоматически из библиотеки:

- **Всё радио** — вся библиотека
- **По жанру** — из тега `genre` (например, Rock, Alternative)
- **По десятилетию** — из тега `year` (например, `2000-е`)

**Пример:**

```
GET /api/stations
```

**Ответ `200 OK`:**

```json
{
  "items": [
    {
      "id": "all",
      "name": "Всё радио",
      "kind": "all",
      "description": "Случайные треки из всей библиотеки",
      "track_count": 532
    },
    {
      "id": "genre:rock",
      "name": "Rock",
      "kind": "genre",
      "description": "Треки жанра Rock",
      "track_count": 210
    },
    {
      "id": "decade:2000",
      "name": "2000-е",
      "kind": "decade",
      "description": "Музыка 2000-е",
      "track_count": 85
    }
  ]
}
```

Для воспроизведения станции используйте `GET /api/radio?station=<id>`.

---

### `GET /api/radio`

Случайный поток треков для режима «радио».

Перемешивает подходящие треки и возвращает очередь. Клиент воспроизводит `items` по порядку и запрашивает новую порцию, когда очередь заканчивается.

**Query-параметры:**

| Параметр    | Тип    | По умолчанию | Описание                                     |
| ----------- | ------ | ------------ | -------------------------------------------- |
| `limit`     | int    | `20`         | Сколько треков вернуть (макс. `50`)          |
| `exclude`   | string | —            | ID треков через запятую, которые не включать |
| `artist_id` | string | —            | Радио по исполнителю                         |
| `station`   | string | —            | ID станции из `/api/stations`                |
| `seed`      | string | —            | ID трека для «похожего» радио                |

**Примеры:**

```
GET /api/radio
GET /api/radio?limit=10
GET /api/radio?artist_id=a1b2c3...
GET /api/radio?station=genre:rock
GET /api/radio?station=decade:2000
GET /api/radio?seed=9f86d081...
GET /api/radio?limit=5&exclude=abc123,def456
```

**Радио по seed** подбирает треки того же исполнителя, жанра или десятилетия.

**Ответ `200 OK`:**

```json
{
  "items": [
    {
      "id": "9f86d081...",
      "title": "Bohemian Rhapsody",
      "artist": "Queen",
      "album": "Greatest Hits",
      "duration": 354.5,
      "format": "flac"
    }
  ],
  "total_available": 180,
  "returned": 20
}
```

**Воспроизведение:** для каждого трека из `items` используйте `GET /api/tracks/{id}/stream?token=...`

**Сценарий iOS:**

1. **Главная:** `GET /api/radio?limit=20` — всё радио
2. **Станции:** `GET /api/stations` → `GET /api/radio?station=genre:rock`
3. **Артист:** `GET /api/radio?artist_id=...` — кнопка «Радио артиста»
4. **Поиск:** `GET /api/search?q=...` → переход к артисту / альбому / треку
5. Когда осталось 2–3 трека: `GET /api/radio?...&exclude=id1,id2,...` — подгрузить следующую порцию

**Ответ `404`:** нет подходящих треков (пустая библиотека, неизвестный seed, пустая станция)

---

### `GET /api/tracks`

Список треков с пагинацией и поиском.

**Query-параметры:**

| Параметр    | Тип    | По умолчанию | Описание                                                                                                                           |
| ----------- | ------ | ------------ | ---------------------------------------------------------------------------------------------------------------------------------- |
| `page`      | int    | `1`          | Номер страницы                                                                                                                     |
| `limit`     | int    | `50`         | Размер страницы (максимум `200`)                                                                                                   |
| `search`    | string | —            | Поиск по title, artist, album, genre                                                                                               |
| `artist_id` | string | —            | Фильтр по ID исполнителя из `/api/artists`                                                                                         |
| `album_id`  | string | —            | Фильтр по ID альбома из `/api/albums`                                                                                              |
| `sort`      | string | —            | Сортировка: `title`, `artist`, `album`, `track_number`, `duration`, `modified_at`. Префикс `-` для убывания, напр. `-track_number` |

**Примеры:**

```
GET /api/tracks?page=1&limit=20&album_id=d4e5f6...&sort=track_number
GET /api/tracks?artist_id=a1b2c3...&sort=-title
```

**Ответ `200 OK`:**

```json
{
  "items": [
    {
      "id": "9f86d081...",
      "title": "Bohemian Rhapsody",
      "artist": "Queen",
      "album": "Greatest Hits",
      "year": 1975,
      "genre": "Rock",
      "duration": 354.5,
      "track_number": 1,
      "disc_number": 1,
      "has_cover": true,
      "size": 8457216,
      "format": "mp3",
      "modified_at": "2024-06-15T10:30:00Z"
    }
  ],
  "total": 128,
  "page": 1,
  "limit": 20,
  "total_pages": 7
}
```

> Поле `id` — стабильный SHA256-хеш от абсолютного пути к файлу.

---

### `GET /api/tracks/{id}`

Информация об одном треке.

**Ответ `200 OK`:** объект `Track` (см. выше)

**Ответ `404 Not Found`:**

```json
{
  "error": true,
  "message": "Track not found"
}
```

---

> Поле `duration` у треков, альбомов и исполнителей — **секунды** (float).

---

### `GET /api/tracks/{id}/stream`

Стриминг аудиофайла. Поддерживаются методы **GET** и **HEAD**.

Поддерживает **HTTP Range Requests** (`Range: bytes=...`).

**Авторизация:** заголовок `Authorization: Bearer <token>` **или** query-параметр `?token=<token>` (для AVPlayer).

```
GET /api/tracks/{id}/stream?token=your-token
Range: bytes=0-1023
```

**Заголовки ответа:**

```
Content-Type: audio/flac
Accept-Ranges: bytes
Content-Length: 25516547
Cache-Control: public, max-age=86400
```

**Ошибки:**

| Статус | Сообщение              |
| ------ | ---------------------- |
| 404    | Track not found        |
| 404    | Track file not found   |
| 500    | Failed to stream track |

---

### `GET /api/tracks/{id}/cover`

Обложка трека.

**Ответ `200 OK`:**

- `Content-Type`: `image/jpeg`, `image/png` и т.д.
- `Cache-Control: public, max-age=3600`
- Тело: бинарные данные изображения

**Ответ `404 Not Found`:**

```json
{
  "error": true,
  "message": "Cover not found"
}
```

---

### `POST /api/rescan`

Полное пересканирование библиотеки.

**Ответ `200 OK`:**

```json
{
  "tracks_found": 128,
  "duration": 2.45
}
```

> Поле `duration` — длительность сканирования в **секундах** (float).

## Интеграция с iOS (AVPlayer)

### Подключение через QR

В панели управления выберите сеть (VPN или домашняя) и отсканируйте QR в приложении. Формат данных:

```json
{ "server": "http://192.168.0.93:8080", "token": "change-me-to-a-secure-token" }
```

### Подключение вручную

| Поле в приложении | Пример                                                |
| ----------------- | ----------------------------------------------------- |
| Сервер            | `http://192.168.0.93:8080`                            |
| Bearer Token      | `change-me-to-a-secure-token` (без префикса `Bearer`) |

### Рекомендуемая структура приложения

| Таб         | API                                                 |
| ----------- | --------------------------------------------------- |
| **Главная** | `GET /api/radio?limit=20`                           |
| **Станции** | `GET /api/stations` → `GET /api/radio?station=<id>` |
| **Поиск**   | `GET /api/search?q=...`                             |

Экран артиста (не таб):

- `GET /api/radio?artist_id=...` — кнопка «Радио артиста»
- `GET /api/albums?artist_id=...` — список альбомов
- `GET /api/albums/{id}/tracks` — треки альбома

### Опциональные поля альбома

Если включено `metadata.enabled`, в ответах `/api/albums` появляются `cover_url`, `genres`, `description`, `metadata_status`. Для загрузки обложки по `cover_url` передавайте заголовок `Authorization: Bearer <token>`.

Обложки исполнителей в API нет — можно использовать обложку первого альбома из `GET /api/albums?artist_id=...`.

При `metadata_status == "pending"` имеет смысл обновлять список альбомов через несколько секунд.

### Базовый URL (для разработки)

```swift
let baseURL = URL(string: "http://192.168.1.100:8080")!
let token = "change-me-to-a-secure-token"
```

### Запросы к API

```swift
var request = URLRequest(url: baseURL.appendingPathComponent("/api/tracks"))
request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
```

### Стриминг

```swift
let streamURL = baseURL.appendingPathComponent("/api/tracks/\(trackID)/stream")

var request = URLRequest(url: streamURL)
request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")

let playerItem = AVPlayerItem(asset: AVURLAsset(url: streamURL, options: [
    "AVURLAssetHTTPHeaderFieldsKey": ["Authorization": "Bearer \(token)"]
]))
let player = AVPlayer(playerItem: playerItem)
```

> AVPlayer автоматически отправляет `Range`-заголовки — сервер их поддерживает. Для быстрого старта воспроизведения можно передать token в URL: `?token=<token>` (альтернатива заголовку Authorization).

### Обложка

```swift
let coverURL = baseURL.appendingPathComponent("/api/tracks/\(trackID)/cover")
// Загрузить через URLSession с Authorization header

// Обложка альбома (в т.ч. из MusicBrainz):
let albumCoverURL = baseURL.appendingPathComponent("/api/albums/\(albumID)/cover")
```

## Автообновление библиотеки

Сервер следит за папкой `music_path` через `fsnotify`:

| Событие            | Действие                         |
| ------------------ | -------------------------------- |
| Добавлен аудиофайл | Индексация нового трека          |
| Изменён аудиофайл  | Переиндексация трека             |
| Удалён аудиофайл   | Удаление из каталога             |
| Переименование     | Remove + Create (debounce 500ms) |
| Новая папка        | Добавляется в watcher рекурсивно |

Ручное пересканирование: `POST /api/rescan`

## Архитектура

```
cmd/
  server/            — консольный запуск
  gui/               — панель управления (127.0.0.1:8099)
internal/
  app/               — общая логика запуска/остановки сервера
  api/               — HTTP handlers, chi router
  auth/              — Bearer token middleware
  config/            — загрузка config.yaml
  cover/             — поиск обложек в папке
  library/           — in-memory хранилище и индексы
  metadata/          — MusicBrainz, SQLite, worker, скачивание обложек
  models/            — доменные структуры
  repository/        — интерфейсы доступа к данным
  scanner/           — сканирование, метаданные, watcher
  service/           — бизнес-логика, обогащение альбомов
  stream/            — стриминг с Range Requests
pkg/
  addresses/         — определение IP для подключения с телефона
  connectqr/         — JSON-полезная нагрузка для QR-кода
  id/                — генерация стабильных ID
data/                  — создаётся при запуске: library.db, covers/
```

### Поток данных

```
HTTP Request
    → api (handlers)
    → service (бизнес-логика, обогащение альбомов)
    → library (in-memory каталог)
    → metadata (SQLite + MusicBrainz, фоновый worker)
    → scanner (индексация файлов, fsnotify)
```

Зависимости внедряются через конструкторы в `cmd/server/main.go`. Глобальных переменных нет.

## Разработка

### Запуск тестов

```bash
go test ./...
```

### Сборка

**Windows:**

```powershell
go build -o bin/server.exe ./cmd/server
go build -o bin\GoMusic.exe .\cmd\gui
```

**Linux / macOS:**

```bash
go build -o bin/server ./cmd/server
go build -o bin/gomusic-gui ./cmd/gui
```

### Зависимости

| Пакет                | Назначение                                        |
| -------------------- | ------------------------------------------------- |
| `go-chi/chi`         | HTTP router                                       |
| `rs/zerolog`         | Логирование                                       |
| `fsnotify/fsnotify`  | Слежение за файловой системой                     |
| `dhowden/tag`        | Чтение аудио-тегов                                |
| `tcolgate/mp3`       | Длительность MP3                                  |
| `skip2/go-qrcode`    | QR-коды в панели управления                       |
| `yaml.v3`            | Конфигурация                                      |
| `modernc.org/sqlite` | SQLite для метаданных альбомов (pure Go, без CGO) |

## Устранение неполадок

### Панель не запускается: `bind: Only one usage of each socket address`

Панель уже запущена (порт `8099` занят). Откройте http://127.0.0.1:8099 или завершите процесс `GoMusic.exe` в диспетчере задач.

### Сервер не стартует: `initial library scan failed`

- Убедитесь, что папка из `music_path` существует
- Проверьте права доступа к папке

### `401 Unauthorized`

- Проверьте заголовок `Authorization: Bearer <token>`
- Убедитесь, что токен совпадает с `config.yaml`

### Трек не воспроизводится в AVPlayer

- Убедитесь, что iOS-устройство в той же сети, что и сервер (или подключено к VPN)
- Используйте правильный адрес: VPN-IP через VPN, домашний IP — в домашней Wi‑Fi
- Передавайте `Authorization` header при стриминге (или `?token=` в URL)
- Проверьте, что файл существует на диске
- На Windows — брандмауэр для порта `8080`; на Linux — `ufw`; на macOS — системный файрвол
- FLAC стартует медленнее MP3; для AVPlayer: `automaticallyWaitsToMinimizeStalling = false`

### Обложка не отображается

- Проверьте `has_cover: true` в объекте трека или альбома
- Для альбомов: дождитесь `metadata_status: success` или добавьте `cover.jpg` в папку альбома
- Загрузка по `cover_url` требует заголовок `Authorization: Bearer <token>`
- Или встройте обложку в теги файла

### После переименования файла старый трек остался

- ID трека зависит от пути — при переименовании создаётся новый трек
- Выполните `POST /api/rescan` для очистки устаревших записей

## Безопасность

Сервер предназначен для **личного использования** в домашней сети:

- Используйте длинный случайный `token`
- Не выставляйте сервер в открытый интернет без TLS и дополнительной защиты
- Токен передаётся в каждом запросе — используйте HTTPS при доступе вне локальной сети

## Лицензия

Проект для личного использования.
