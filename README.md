# Stratus

Self-hosted file storage system with WebDAV support, authentication, and web interface.

## Features

- File management (upload, download, delete)
- User authentication (login, register)
- WebDAV support
- Modern React UI
- Admin panel
- Activity tracking
- Docker deployment

## Tech Stack

- **Backend**: Go, PostgreSQL
- **Frontend**: React, TypeScript, Tailwind CSS, Vite
- **Infrastructure**: Docker, Docker Compose, Nginx

## Quick Start

### With Docker Compose

```bash
git clone <repository-url>
cd Stratus
docker-compose up -d
```

Access:
- Frontend: http://localhost
- API: http://localhost:3000

### Manual Setup

**Backend:**
```bash
cd backend
go mod download
go run main.go
```

**Frontend:**
```bash
cd frontend
npm install
npm run dev
```

## Project Structure

```
Stratus/
├── backend/          # Go API server
├── frontend/         # React web app
├── database/         # DB schemas
└── docker-compose.yml
```

## API Endpoints

- `POST /auth/register` - Register user
- `POST /auth/login` - Login
- `GET /files` - List files
- `POST /files/upload` - Upload file
- `DELETE /files/:id` - Delete file
- `GET /webdav` - WebDAV endpoint

## Environment Variables

```env
DB_HOST=postgres
DB_PORT=5432
DB_USER=stratus
DB_PASSWORD=password
DB_NAME=stratus
SERVER_PORT=3000
VITE_API_URL=http://localhost:3000
```

## Development

```bash
# Backend
cd backend
go run main.go

# Frontend
cd frontend
pnpm install
pnpm dev
```

## Build

```bash
# Backend
cd backend
go build -o stratus-backend

# Frontend
cd frontend
pnpm build
```

## License

Apache License 2.0 - See [LICENSE](LICENSE)
