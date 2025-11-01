# Restaurant Site - Complete Application

A full-stack restaurant management system with admin panel, secure authentication, and media export.

## Features

### Customer-Facing
- Browse menu with categories
- Add items to cart and place orders
- View gallery and reviews
- Subscribe to newsletter

### Admin Features
- Secure JWT authentication with refresh tokens
- Session management (view/revoke active sessions)
- Order management
- Menu editing
- Admin invitation system
- Password reset flow
- Audit logging
- Data export (JSON)
- Media export (ZIP or S3)

## Tech Stack

**Backend:**
- Go 1.20+
- PostgreSQL
- JWT authentication
- AWS SDK (for S3 exports)

**Frontend:**
- React 18
- React Router
- Axios

## Quick Start

### 1. Prerequisites
- Docker and Docker Compose
- Go 1.20+
- Node.js 18+
- PostgreSQL 15+

### 2. Setup

```bash
# Copy environment variables
cp .env.example .env

# Edit .env with your configuration
# At minimum, set JWT_SECRET to a strong random value

# Start services with Docker
make up

# Or manually:
docker-compose up --build
```

### 3. Initialize Database

```bash
# Run migrations
make migrate

# Create first admin user
make create-admin RESTAURANT_ID=1 ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=strongpass123 ADMIN_ROLE=owner
```

### 4. Access Application

- Frontend: http://localhost:3000
- Backend API: http://localhost:8080
- Admin panel: http://localhost:3000/restaurant/1/admin

## Development

### Running Locally (without Docker)

**Backend:**
```bash
cd backend
go mod download
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/resto?sslmode=disable"
export JWT_SECRET="your-secret-key"
go run .
```

**Frontend:**
```bash
cd frontend
npm install
npm start
```

### Database Migrations

```bash
# Run all migrations
./scripts/run_migrations.sh

# Or manually with psql
psql $DATABASE_URL -f db/migrations.sql
psql $DATABASE_URL -f db/admin_onboarding_migrations.sql
psql $DATABASE_URL -f db/password_reset_migration.sql
psql $DATABASE_URL -f db/refresh_tokens_migration.sql
```

### Creating Admin Users

```bash
# Using the script
go run scripts/create_admin.go --restaurant 1 --email admin@example.com --password 'SecurePass123' --role owner

# Using Makefile
make create-admin RESTAURANT_ID=1 ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=pass123 ADMIN_ROLE=owner
```

## API Endpoints

### Public Endpoints
- `GET /api/restaurants/:id` - Get restaurant data
- `POST /api/orders/:id` - Place order
- `POST /api/subscribe/:id` - Subscribe to newsletter
- `POST /api/reviews/:id` - Submit review

### Auth Endpoints
- `POST /api/login` - Login (returns JWT + refresh token cookie)
- `POST /api/refresh` - Refresh access token
- `POST /api/logout` - Logout and revoke session
- `GET /api/verify` - Verify token

### Admin Endpoints (require authentication)
- `GET /api/admin/orders/:id` - List orders
- `POST /api/menus/:id` - Update menus
- `POST /api/restaurants_patch/:id` - Update restaurant info
- `POST /api/admin/invite/:id` - Invite admin (owner only)
- `POST /api/admin/invite/accept` - Accept invitation
- `POST /api/admin/password_reset/request` - Request password reset
- `POST /api/admin/password_reset/confirm` - Confirm password reset
- `GET /api/admin/sessions/:id` - List sessions
- `POST /api/admin/sessions/revoke` - Revoke session
- `POST /api/admin/sessions/revoke_all` - Revoke all other sessions
- `GET /api/admin/export/:id` - Export data (JSON)
- `GET/POST /api/admin/export_media/:id` - Export media (ZIP/S3)
- `GET /api/admin/audit/:id` - View audit log

## Security Features

1. **JWT with Refresh Tokens**: Short-lived access tokens (15min) + HTTP-only refresh cookies (30 days)
2. **Token Rotation**: Refresh tokens are rotated on each use
3. **Session Management**: View and revoke active sessions
4. **Password Reset**: Time-limited tokens (1 hour expiry)
5. **Admin Invitations**: Secure invitation flow with 72-hour expiry
6. **Audit Logging**: All admin actions logged with IP addresses
7. **CSRF Protection**: SameSite cookies
8. **bcrypt**: Password hashing with cost factor 10

## Environment Variables

See `.env.example` for all available variables. Key ones:

```bash
DATABASE_URL=postgres://postgres:postgres@localhost:5432/resto?sslmode=disable
JWT_SECRET=change-me-to-a-strong-secret
FRONTEND_URL=http://localhost:3000
ALLOW_ORIGIN=http://localhost:3000
ENV=development
ALLOW_INSECURE_COOKIES=1  # Set to 0 in production

# SMTP (optional, for emails)
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=user@example.com
SMTP_PASS=password
SMTP_FROM=notifications@example.com

# AWS (for S3 exports)
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=your-key
AWS_SECRET_ACCESS_KEY=your-secret
```

## Production Deployment

1. Set `ENV=production` in environment
2. Set `ALLOW_INSECURE_COOKIES=0`
3. Use strong `JWT_SECRET` (32+ random characters)
4. Configure proper CORS origins
5. Enable HTTPS
6. Set up regular database backups
7. Configure SMTP for email notifications
8. Run cleanup worker for expired tokens

```bash
# Start cleanup worker
docker-compose up cleanup-worker
```

## Maintenance

### Cleanup Old Tokens

```bash
# Manual cleanup
go run scripts/cleanup_refresh_tokens.go --retention 30

# Or via Makefile
make cleanup-sessions
```

### Backup Database

```bash
pg_dump $DATABASE_URL > backup.sql
```

## Troubleshooting

**"Database connection failed"**
- Check DATABASE_URL is correct
- Ensure PostgreSQL is running
- Verify network connectivity

**"Invalid token"**
- Check JWT_SECRET matches between restarts
- Token may have expired (15min for access tokens)
- Try refreshing the session

**"SMTP errors"**
- SMTP is optional; app works without it
- Check SMTP credentials if email features needed

**"CORS errors"**
- Verify ALLOW_ORIGIN matches your frontend URL
- Check that credentials: true is set in frontend API calls

## License

MIT

## Support

For issues, please open a GitHub issue with:
- Error messages
- Steps to reproduce
- Environment details (OS, Go version, etc.)

