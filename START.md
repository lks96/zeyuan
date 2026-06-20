# Temu Tools Startup

## Windows

Prepare the environment for the first time:

```powershell
.\scripts\setup-dev.ps1
```

Start backend and frontend:

```powershell
.\scripts\start-dev.ps1
```

Restart backend and frontend in the background:

```powershell
.\scripts\restart-dev.ps1 -SkipInstall -SkipMigrate
```

If PowerShell script execution is blocked, run:

```cmd
scripts\start-dev.cmd
```

Restart from cmd:

```cmd
scripts\restart-dev.cmd -SkipInstall -SkipMigrate
```

## macOS / Linux

Prepare the environment for the first time:

```bash
bash ./scripts/setup-dev.sh
```

Start backend and frontend:

```bash
bash ./scripts/start-dev.sh
```

Start backend and frontend in the background:

```bash
bash ./scripts/start-dev.sh --background
```

Restart backend and frontend in the background:

```bash
SKIP_INSTALL=1 SKIP_MIGRATE=1 bash ./scripts/restart-dev.sh
```

## Environment

The scripts check `backend/.env`. If it does not exist, they copy `backend/.env.example` to `backend/.env` and ask you to fill:

```text
DB_PASSWORD=your_database_password
```

After editing `backend/.env`, run the setup or start script again.

Default URLs:

```text
Frontend: http://localhost:5173
Backend API: http://localhost:8080
```

Default accounts:

```text
admin / admin123
operator_a / operator123
```

## Options

Windows, skip dependency install:

```powershell
.\scripts\start-dev.ps1 -SkipInstall
```

Windows, skip database migration:

```powershell
.\scripts\start-dev.ps1 -SkipMigrate
```

macOS / Linux, skip dependency install:

```bash
SKIP_INSTALL=1 bash ./scripts/start-dev.sh
```

macOS / Linux, skip database migration:

```bash
SKIP_MIGRATE=1 bash ./scripts/start-dev.sh
```
