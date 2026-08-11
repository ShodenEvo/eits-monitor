from contextlib import asynccontextmanager
from datetime import datetime, timedelta, timezone
import hashlib
import json
import hmac
import secrets
from typing import Annotated

import jwt
from apscheduler.schedulers.background import BackgroundScheduler
from fastapi import Cookie, Depends, FastAPI, Header, HTTPException, Response, status
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field, field_validator
from pydantic_settings import BaseSettings, SettingsConfigDict
from pwdlib import PasswordHash
from sqlalchemy import Boolean, DateTime, Float, ForeignKey, Integer, String, BigInteger, Text, UniqueConstraint, create_engine, delete, func, select, text
from sqlalchemy.orm import DeclarativeBase, Mapped, Session, mapped_column, relationship, sessionmaker


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file='.env', extra='ignore')
    database_url: str
    secret_key: str
    admin_username: str = 'admin'
    admin_password: str
    agent_enrollment_token: str
    metric_retention_days: int = 30
    cookie_secure: bool = False

settings = Settings()
engine = create_engine(settings.database_url, pool_pre_ping=True)
SessionLocal = sessionmaker(engine, expire_on_commit=False)
password_hash = PasswordHash.recommended()


class Base(DeclarativeBase):
    pass


class User(Base):
    __tablename__ = 'users'
    id: Mapped[int] = mapped_column(primary_key=True)
    username: Mapped[str] = mapped_column(String(100), unique=True, index=True)
    password_hash: Mapped[str] = mapped_column(String(500))
    is_active: Mapped[bool] = mapped_column(Boolean, default=True)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=lambda: datetime.now(timezone.utc))


class Device(Base):
    __tablename__ = 'devices'
    id: Mapped[int] = mapped_column(primary_key=True)
    agent_id: Mapped[str] = mapped_column(String(64), unique=True, index=True)
    agent_secret_hash: Mapped[str] = mapped_column(String(128))
    name: Mapped[str] = mapped_column(String(200))
    hostname: Mapped[str] = mapped_column(String(200), default='')
    os: Mapped[str] = mapped_column(String(100), default='')
    architecture: Mapped[str] = mapped_column(String(100), default='')
    agent_version: Mapped[str] = mapped_column(String(50), default='')
    last_seen: Mapped[datetime | None] = mapped_column(DateTime(timezone=True), nullable=True)
    enrolled_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=lambda: datetime.now(timezone.utc))
    warning_disk_percent: Mapped[float] = mapped_column(Float, default=80.0)
    critical_disk_percent: Mapped[float] = mapped_column(Float, default=90.0)
    metrics: Mapped[list['Metric']] = relationship(cascade='all, delete-orphan')
    disks: Mapped[list['DiskMetric']] = relationship(cascade='all, delete-orphan')
    port_checks: Mapped[list['PortCheck']] = relationship(cascade='all, delete-orphan')
    inventory: Mapped['DeviceInventory | None'] = relationship(cascade='all, delete-orphan', uselist=False)
    running_processes: Mapped[list['RunningProcess']] = relationship(cascade='all, delete-orphan')
    process_monitors: Mapped[list['ProcessMonitor']] = relationship(cascade='all, delete-orphan')


class DeviceInventory(Base):
    __tablename__ = 'device_inventory'
    id: Mapped[int] = mapped_column(primary_key=True)
    device_id: Mapped[int] = mapped_column(ForeignKey('devices.id', ondelete='CASCADE'), unique=True, index=True)
    collected_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), index=True)
    manufacturer: Mapped[str] = mapped_column(String(200), default='')
    model: Mapped[str] = mapped_column(String(200), default='')
    serial_number: Mapped[str] = mapped_column(String(200), default='')
    device_type: Mapped[str] = mapped_column(String(50), default='unknown')
    os_name: Mapped[str] = mapped_column(String(200), default='')
    os_version: Mapped[str] = mapped_column(String(100), default='')
    os_build: Mapped[str] = mapped_column(String(100), default='')
    kernel_version: Mapped[str] = mapped_column(String(200), default='')
    last_os_update: Mapped[str] = mapped_column(String(200), default='')
    cpu_vendor: Mapped[str] = mapped_column(String(100), default='')
    cpu_model: Mapped[str] = mapped_column(String(300), default='')
    cpu_physical_cores: Mapped[int] = mapped_column(Integer, default=0)
    cpu_logical_processors: Mapped[int] = mapped_column(Integer, default=0)
    total_memory_bytes: Mapped[int] = mapped_column(BigInteger, default=0)
    bios_version: Mapped[str] = mapped_column(String(200), default='')
    gpus_json: Mapped[str] = mapped_column(Text, default='[]')


class Metric(Base):
    __tablename__ = 'metrics'
    id: Mapped[int] = mapped_column(primary_key=True)
    device_id: Mapped[int] = mapped_column(ForeignKey('devices.id'), index=True)
    recorded_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), index=True)
    cpu_percent: Mapped[float] = mapped_column(Float)
    memory_percent: Mapped[float] = mapped_column(Float)
    memory_total: Mapped[int] = mapped_column(BigInteger)
    memory_used: Mapped[int] = mapped_column(BigInteger)
    uptime_seconds: Mapped[int] = mapped_column(BigInteger)
    network_sent: Mapped[int] = mapped_column(BigInteger, default=0)
    network_recv: Mapped[int] = mapped_column(BigInteger, default=0)


class DiskMetric(Base):
    __tablename__ = 'disk_metrics'
    id: Mapped[int] = mapped_column(primary_key=True)
    device_id: Mapped[int] = mapped_column(ForeignKey('devices.id'), index=True)
    recorded_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), index=True)
    mountpoint: Mapped[str] = mapped_column(String(500))
    filesystem: Mapped[str] = mapped_column(String(100), default='')
    total: Mapped[int] = mapped_column(BigInteger)
    used: Mapped[int] = mapped_column(BigInteger)
    free: Mapped[int] = mapped_column(BigInteger)
    percent: Mapped[float] = mapped_column(Float)


class PortCheck(Base):
    __tablename__ = 'port_checks'
    id: Mapped[int] = mapped_column(primary_key=True)
    device_id: Mapped[int] = mapped_column(ForeignKey('devices.id'), index=True)
    name: Mapped[str] = mapped_column(String(200))
    host: Mapped[str] = mapped_column(String(255))
    port: Mapped[int] = mapped_column(Integer)
    timeout_seconds: Mapped[int] = mapped_column(Integer, default=3)
    protocol: Mapped[str] = mapped_column(String(10), default='tcp')
    udp_payload: Mapped[str] = mapped_column(String(1000), default='')
    expect_response: Mapped[bool] = mapped_column(Boolean, default=False)
    enabled: Mapped[bool] = mapped_column(Boolean, default=True)


class PortResult(Base):
    __tablename__ = 'port_results'
    id: Mapped[int] = mapped_column(primary_key=True)
    device_id: Mapped[int] = mapped_column(ForeignKey('devices.id'), index=True)
    check_id: Mapped[int] = mapped_column(ForeignKey('port_checks.id', ondelete='CASCADE'), index=True)
    recorded_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), index=True)
    is_up: Mapped[bool] = mapped_column(Boolean)
    latency_ms: Mapped[float] = mapped_column(Float, default=0)
    error: Mapped[str] = mapped_column(String(500), default='')


class RunningProcess(Base):
    __tablename__ = 'running_processes'
    __table_args__ = (UniqueConstraint('device_id', 'pid', name='uq_running_process_device_pid'),)
    id: Mapped[int] = mapped_column(primary_key=True)
    device_id: Mapped[int] = mapped_column(ForeignKey('devices.id', ondelete='CASCADE'), index=True)
    collected_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), index=True)
    pid: Mapped[int] = mapped_column(Integer)
    name: Mapped[str] = mapped_column(String(300), index=True)
    memory_bytes: Mapped[int] = mapped_column(BigInteger, default=0)


class ProcessMonitor(Base):
    __tablename__ = 'process_monitors'
    __table_args__ = (UniqueConstraint('device_id', 'name', name='uq_process_monitor_device_name'),)
    id: Mapped[int] = mapped_column(primary_key=True)
    device_id: Mapped[int] = mapped_column(ForeignKey('devices.id', ondelete='CASCADE'), index=True)
    name: Mapped[str] = mapped_column(String(300))
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), default=lambda: datetime.now(timezone.utc))


def db_session():
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()

DB = Annotated[Session, Depends(db_session)]


def create_token(user_id: int) -> str:
    now = datetime.now(timezone.utc)
    return jwt.encode({'sub': str(user_id), 'iat': now, 'exp': now + timedelta(hours=12)}, settings.secret_key, algorithm='HS256')


def current_user(db: DB, eits_session: Annotated[str | None, Cookie()] = None) -> User:
    if not eits_session:
        raise HTTPException(status_code=401, detail='Not authenticated')
    try:
        payload = jwt.decode(eits_session, settings.secret_key, algorithms=['HS256'])
        user = db.get(User, int(payload['sub']))
    except Exception:
        raise HTTPException(status_code=401, detail='Invalid or expired session')
    if not user or not user.is_active:
        raise HTTPException(status_code=401, detail='Inactive user')
    return user


def authenticate_agent(db: Session, agent_id: str, agent_secret: str) -> Device:
    device = db.scalar(select(Device).where(Device.agent_id == agent_id))
    if not device:
        raise HTTPException(status_code=401, detail='Unknown agent')
    digest = hashlib.sha256(agent_secret.encode()).hexdigest()
    if not hmac.compare_digest(digest, device.agent_secret_hash):
        raise HTTPException(status_code=401, detail='Invalid agent credential')
    return device


class LoginRequest(BaseModel):
    username: str
    password: str

class EnrollRequest(BaseModel):
    enrollment_token: str
    name: str = Field(min_length=1, max_length=200)
    hostname: str = ''
    os: str = ''
    architecture: str = ''
    agent_version: str = ''

class AgentIdentity(BaseModel):
    agent_id: str
    agent_secret: str

class DiskIn(BaseModel):
    mountpoint: str
    filesystem: str = ''
    total: int
    used: int
    free: int
    percent: float

class PortResultIn(BaseModel):
    check_id: int
    is_up: bool
    latency_ms: float = 0
    error: str = ''

class ProcessIn(BaseModel):
    pid: int = Field(ge=0)
    name: str = Field(min_length=1, max_length=300)
    memory_bytes: int = Field(default=0, ge=0)

class MetricsIn(BaseModel):
    recorded_at: datetime
    hostname: str = ''
    os: str = ''
    architecture: str = ''
    agent_version: str = ''
    cpu_percent: float
    memory_percent: float
    memory_total: int
    memory_used: int
    uptime_seconds: int
    network_sent: int = 0
    network_recv: int = 0
    disks: list[DiskIn] = Field(default_factory=list)
    port_results: list[PortResultIn] = Field(default_factory=list)
    processes: list[ProcessIn] = Field(default_factory=list, max_length=5000)

    @field_validator('disks', 'port_results', 'processes', mode='before')
    @classmethod
    def null_lists_become_empty(cls, value):
        return [] if value is None else value

class GPUIn(BaseModel):
    vendor: str = ''
    model: str = ''
    memory_bytes: int = 0
    driver_version: str = ''

class InventoryIn(BaseModel):
    collected_at: datetime
    manufacturer: str = ''
    model: str = ''
    serial_number: str = ''
    device_type: str = 'unknown'
    os_name: str = ''
    os_version: str = ''
    os_build: str = ''
    kernel_version: str = ''
    last_os_update: str = ''
    cpu_vendor: str = ''
    cpu_model: str = ''
    cpu_physical_cores: int = 0
    cpu_logical_processors: int = 0
    total_memory_bytes: int = 0
    bios_version: str = ''
    gpus: list[GPUIn] = Field(default_factory=list)

class PortCheckCreate(BaseModel):
    name: str = Field(min_length=1, max_length=200)
    host: str = Field(min_length=1, max_length=255)
    port: int = Field(ge=1, le=65535)
    protocol: str = Field(default='tcp', pattern='^(tcp|udp)$')
    timeout_seconds: int = Field(default=3, ge=1, le=30)
    udp_payload: str = Field(default='', max_length=1000)
    expect_response: bool = False

    @field_validator('protocol', mode='before')
    @classmethod
    def normalise_protocol(cls, value):
        return str(value or 'tcp').lower()

class ThresholdUpdate(BaseModel):
    warning_disk_percent: float = Field(ge=1, le=99)
    critical_disk_percent: float = Field(ge=2, le=100)

class ProcessMonitorCreate(BaseModel):
    name: str = Field(min_length=1, max_length=300)

    @field_validator('name')
    @classmethod
    def clean_name(cls, value):
        return value.strip()


def device_health(device: Device, latest_disks: list[DiskMetric], failed_ports: int, failed_processes: list[str]) -> tuple[str, list[str]]:
    if not device.last_seen:
        return 'unknown', ['Waiting for the first agent report']
    age = (datetime.now(timezone.utc) - device.last_seen).total_seconds()
    if age > 180:
        return 'offline', [f'Agent has not reported for {max(3, round(age / 60))} minutes']

    critical_reasons = []
    warning_reasons = []
    critical_disks = [disk for disk in latest_disks if disk.percent >= device.critical_disk_percent]
    warning_disks = [disk for disk in latest_disks if device.warning_disk_percent <= disk.percent < device.critical_disk_percent]
    if critical_disks:
        critical_reasons.append(f'{len(critical_disks)} disk{"s" if len(critical_disks) != 1 else ""} above critical threshold')
    if failed_processes:
        critical_reasons.append(f'{len(failed_processes)} monitored process{"es" if len(failed_processes) != 1 else ""} not running')
    if failed_ports >= 2:
        critical_reasons.append(f'{failed_ports} network checks failing')
    elif failed_ports == 1:
        warning_reasons.append('1 network check failing')
    if warning_disks:
        warning_reasons.append(f'{len(warning_disks)} disk{"s" if len(warning_disks) != 1 else ""} above warning threshold')
    if age > 90:
        warning_reasons.append('Agent reporting is delayed')

    if critical_reasons:
        return 'critical', critical_reasons + warning_reasons
    if warning_reasons:
        return 'warning', warning_reasons
    return 'healthy', []


def latest_disks(db: Session, device_id: int) -> list[DiskMetric]:
    latest_time = db.scalar(select(func.max(DiskMetric.recorded_at)).where(DiskMetric.device_id == device_id))
    if not latest_time:
        return []
    return list(db.scalars(select(DiskMetric).where(DiskMetric.device_id == device_id, DiskMetric.recorded_at == latest_time)))


def serialise_device(db: Session, d: Device):
    metric = db.scalar(select(Metric).where(Metric.device_id == d.id).order_by(Metric.recorded_at.desc()).limit(1))
    disks = latest_disks(db, d.id)
    enabled_check_ids = select(PortCheck.id).where(PortCheck.device_id == d.id, PortCheck.enabled.is_(True))
    failed_ports = db.scalar(
        select(func.count()).select_from(PortResult).where(
            PortResult.device_id == d.id,
            PortResult.check_id.in_(enabled_check_ids),
            PortResult.id.in_(select(func.max(PortResult.id)).where(
                PortResult.device_id == d.id,
                PortResult.check_id.in_(enabled_check_ids),
            ).group_by(PortResult.check_id)),
            PortResult.is_up.is_(False),
        )
    ) or 0
    monitored_processes = list(db.scalars(select(ProcessMonitor).where(ProcessMonitor.device_id == d.id).order_by(ProcessMonitor.name)))
    running_names = {name.casefold() for name in db.scalars(select(RunningProcess.name).where(RunningProcess.device_id == d.id))}
    failed_process_names = [monitor.name for monitor in monitored_processes if monitor.name.casefold() not in running_names]
    status, health_reasons = device_health(d, disks, failed_ports, failed_process_names)
    inventory = d.inventory
    inventory_data = None if not inventory else {
        'collected_at': inventory.collected_at, 'manufacturer': inventory.manufacturer,
        'model': inventory.model, 'serial_number': inventory.serial_number,
        'device_type': inventory.device_type, 'os_name': inventory.os_name,
        'os_version': inventory.os_version, 'os_build': inventory.os_build,
        'kernel_version': inventory.kernel_version, 'last_os_update': inventory.last_os_update,
        'cpu_vendor': inventory.cpu_vendor, 'cpu_model': inventory.cpu_model,
        'cpu_physical_cores': inventory.cpu_physical_cores,
        'cpu_logical_processors': inventory.cpu_logical_processors,
        'total_memory_bytes': inventory.total_memory_bytes,
        'bios_version': inventory.bios_version, 'gpus': json.loads(inventory.gpus_json or '[]'),
    }
    return {
        'id': d.id, 'name': d.name, 'hostname': d.hostname, 'os': d.os,
        'architecture': d.architecture, 'agent_version': d.agent_version,
        'last_seen': d.last_seen, 'status': status,
        'warning_disk_percent': d.warning_disk_percent,
        'critical_disk_percent': d.critical_disk_percent,
        'metric': None if not metric else {
            'recorded_at': metric.recorded_at, 'cpu_percent': metric.cpu_percent,
            'memory_percent': metric.memory_percent, 'memory_total': metric.memory_total,
            'memory_used': metric.memory_used, 'uptime_seconds': metric.uptime_seconds,
            'network_sent': metric.network_sent, 'network_recv': metric.network_recv,
        },
        'disks': [{'mountpoint': x.mountpoint, 'filesystem': x.filesystem, 'total': x.total, 'used': x.used, 'free': x.free, 'percent': x.percent} for x in disks],
        'failed_ports': failed_ports, 'failed_processes': len(failed_process_names),
        'failed_process_names': failed_process_names, 'total_failed_checks': failed_ports + len(failed_process_names),
        'health_reasons': health_reasons, 'inventory': inventory_data,
    }


def cleanup_metrics():
    cutoff = datetime.now(timezone.utc) - timedelta(days=settings.metric_retention_days)
    with SessionLocal() as db:
        db.execute(delete(PortResult).where(PortResult.recorded_at < cutoff))
        db.execute(delete(DiskMetric).where(DiskMetric.recorded_at < cutoff))
        db.execute(delete(Metric).where(Metric.recorded_at < cutoff))
        db.commit()

scheduler = BackgroundScheduler(timezone='UTC')

@asynccontextmanager
async def lifespan(app: FastAPI):
    Base.metadata.create_all(engine)
    # Lightweight forward migration for installations created before UDP support.
    with engine.begin() as connection:
        connection.execute(text("ALTER TABLE port_checks ADD COLUMN IF NOT EXISTS protocol VARCHAR(10) NOT NULL DEFAULT 'tcp'"))
        connection.execute(text("ALTER TABLE port_checks ADD COLUMN IF NOT EXISTS udp_payload VARCHAR(1000) NOT NULL DEFAULT ''"))
        connection.execute(text("ALTER TABLE port_checks ADD COLUMN IF NOT EXISTS expect_response BOOLEAN NOT NULL DEFAULT FALSE"))
        # Ensure deleting a monitoring check also removes its historical results.
        # PostgreSQL does not update existing foreign keys through create_all(),
        # therefore migrate the constraint explicitly for existing installations.
        connection.execute(text("""
            ALTER TABLE port_results
            DROP CONSTRAINT IF EXISTS port_results_check_id_fkey
        """))
        connection.execute(text("""
            ALTER TABLE port_results
            ADD CONSTRAINT port_results_check_id_fkey
            FOREIGN KEY (check_id) REFERENCES port_checks(id) ON DELETE CASCADE
        """))
    with SessionLocal() as db:
        user = db.scalar(select(User).where(User.username == settings.admin_username))
        if not user:
            db.add(User(username=settings.admin_username, password_hash=password_hash.hash(settings.admin_password)))
            db.commit()
    scheduler.add_job(cleanup_metrics, 'interval', hours=24, id='metric_cleanup', replace_existing=True)
    scheduler.start()
    yield
    scheduler.shutdown(wait=False)

app = FastAPI(title='EITS Monitor API', version='0.4.0-alpha.1', lifespan=lifespan)
app.add_middleware(CORSMiddleware, allow_origins=[], allow_credentials=True, allow_methods=['*'], allow_headers=['*'])

@app.get('/api/health')
def health():
    return {'status': 'ok', 'time': datetime.now(timezone.utc)}

@app.post('/api/auth/login')
def login(payload: LoginRequest, response: Response, db: DB):
    user = db.scalar(select(User).where(User.username == payload.username))
    if not user or not password_hash.verify(payload.password, user.password_hash):
        raise HTTPException(status_code=401, detail='Invalid username or password')
    response.set_cookie('eits_session', create_token(user.id), httponly=True, secure=settings.cookie_secure, samesite='lax', max_age=43200, path='/')
    return {'username': user.username}

@app.post('/api/auth/logout')
def logout(response: Response):
    response.delete_cookie('eits_session', path='/')
    return {'ok': True}

@app.get('/api/auth/me')
def me(user: Annotated[User, Depends(current_user)]):
    return {'id': user.id, 'username': user.username}

@app.post('/api/agent/enroll', response_model=AgentIdentity)
def enroll(payload: EnrollRequest, db: DB):
    if not secrets.compare_digest(payload.enrollment_token, settings.agent_enrollment_token):
        raise HTTPException(status_code=403, detail='Invalid enrollment token')
    agent_id = secrets.token_hex(16)
    agent_secret = secrets.token_urlsafe(48)
    db.add(Device(agent_id=agent_id, agent_secret_hash=hashlib.sha256(agent_secret.encode()).hexdigest(), name=payload.name, hostname=payload.hostname, os=payload.os, architecture=payload.architecture, agent_version=payload.agent_version))
    db.commit()
    return AgentIdentity(agent_id=agent_id, agent_secret=agent_secret)

@app.post('/api/agent/inventory', status_code=204)
def ingest_inventory(payload: InventoryIn, db: DB, x_agent_id: Annotated[str, Header()], x_agent_secret: Annotated[str, Header()]):
    device = authenticate_agent(db, x_agent_id, x_agent_secret)
    values = payload.model_dump(exclude={'gpus'})
    values['gpus_json'] = json.dumps([gpu.model_dump() for gpu in payload.gpus])
    inventory = db.scalar(select(DeviceInventory).where(DeviceInventory.device_id == device.id))
    if inventory:
        for key, value in values.items():
            setattr(inventory, key, value)
    else:
        db.add(DeviceInventory(device_id=device.id, **values))
    db.commit()
    return Response(status_code=204)

@app.get('/api/agent/config')
def agent_config(db: DB, x_agent_id: Annotated[str, Header()], x_agent_secret: Annotated[str, Header()]):
    device = authenticate_agent(db, x_agent_id, x_agent_secret)
    checks = list(db.scalars(select(PortCheck).where(PortCheck.device_id == device.id, PortCheck.enabled.is_(True))))
    return {'revision': max([c.id for c in checks], default=0), 'port_checks': [{'id': c.id, 'name': c.name, 'host': c.host, 'port': c.port, 'protocol': c.protocol, 'timeout_seconds': c.timeout_seconds, 'udp_payload': c.udp_payload, 'expect_response': c.expect_response} for c in checks]}

@app.post('/api/agent/metrics')
def ingest(payload: MetricsIn, db: DB, x_agent_id: Annotated[str, Header()], x_agent_secret: Annotated[str, Header()]):
    device = authenticate_agent(db, x_agent_id, x_agent_secret)
    now = datetime.now(timezone.utc)
    device.last_seen = now
    device.hostname, device.os, device.architecture, device.agent_version = payload.hostname, payload.os, payload.architecture, payload.agent_version
    db.add(Metric(device_id=device.id, recorded_at=payload.recorded_at, cpu_percent=payload.cpu_percent, memory_percent=payload.memory_percent, memory_total=payload.memory_total, memory_used=payload.memory_used, uptime_seconds=payload.uptime_seconds, network_sent=payload.network_sent, network_recv=payload.network_recv))
    db.add_all([DiskMetric(device_id=device.id, recorded_at=payload.recorded_at, **d.model_dump()) for d in payload.disks])
    valid_ids = set(db.scalars(select(PortCheck.id).where(PortCheck.device_id == device.id)))
    db.add_all([PortResult(device_id=device.id, recorded_at=payload.recorded_at, **r.model_dump()) for r in payload.port_results if r.check_id in valid_ids])
    # Process inventory is latest-state data, not time-series data. Replace it
    # atomically on each report to prevent unbounded database growth.
    db.execute(delete(RunningProcess).where(RunningProcess.device_id == device.id))
    seen_pids = set()
    processes = []
    for process in payload.processes:
        if process.pid in seen_pids:
            continue
        seen_pids.add(process.pid)
        processes.append(RunningProcess(device_id=device.id, collected_at=payload.recorded_at, **process.model_dump()))
    db.add_all(processes)
    db.commit()
    return {'ok': True}

@app.get('/api/dashboard')
def dashboard(db: DB, _: Annotated[User, Depends(current_user)]):
    devices = [serialise_device(db, d) for d in db.scalars(select(Device).order_by(Device.name))]
    counts = {x: 0 for x in ['healthy', 'warning', 'critical', 'offline', 'unknown']}
    for d in devices: counts[d['status']] += 1
    return {'counts': counts, 'total': len(devices), 'devices': devices}

@app.get('/api/devices/{device_id}')
def get_device(device_id: int, db: DB, _: Annotated[User, Depends(current_user)]):
    device = db.get(Device, device_id)
    if not device: raise HTTPException(404, 'Device not found')
    result = serialise_device(db, device)
    result['port_checks'] = []
    for c in db.scalars(select(PortCheck).where(PortCheck.device_id == device.id).order_by(PortCheck.name)):
        latest = db.scalar(select(PortResult).where(PortResult.check_id == c.id).order_by(PortResult.recorded_at.desc()).limit(1))
        result['port_checks'].append({'id': c.id, 'name': c.name, 'host': c.host, 'port': c.port, 'protocol': c.protocol, 'timeout_seconds': c.timeout_seconds, 'udp_payload': c.udp_payload, 'expect_response': c.expect_response, 'enabled': c.enabled, 'latest': None if not latest else {'recorded_at': latest.recorded_at, 'is_up': latest.is_up, 'latency_ms': latest.latency_ms, 'error': latest.error}})
    history = list(db.scalars(select(Metric).where(Metric.device_id == device.id).order_by(Metric.recorded_at.desc()).limit(120)))
    result['history'] = [{'recorded_at': m.recorded_at, 'cpu_percent': m.cpu_percent, 'memory_percent': m.memory_percent} for m in reversed(history)]
    running = list(db.scalars(select(RunningProcess).where(RunningProcess.device_id == device.id).order_by(RunningProcess.name, RunningProcess.pid)))
    monitors = list(db.scalars(select(ProcessMonitor).where(ProcessMonitor.device_id == device.id).order_by(ProcessMonitor.name)))
    running_names = {process.name.casefold() for process in running}
    result['processes'] = [{'pid': process.pid, 'name': process.name, 'memory_bytes': process.memory_bytes, 'collected_at': process.collected_at} for process in running]
    result['process_monitors'] = [{'id': monitor.id, 'name': monitor.name, 'running': monitor.name.casefold() in running_names} for monitor in monitors]
    return result

@app.delete('/api/devices/{device_id}', status_code=204)
def delete_device(device_id: int, db: DB, _: Annotated[User, Depends(current_user)]):
    device = db.get(Device, device_id)
    if not device:
        raise HTTPException(404, 'Device not found')
    check_ids = select(PortCheck.id).where(PortCheck.device_id == device_id)
    db.execute(delete(PortResult).where(PortResult.device_id == device_id))
    db.execute(delete(PortCheck).where(PortCheck.id.in_(check_ids)))
    db.execute(delete(RunningProcess).where(RunningProcess.device_id == device_id))
    db.execute(delete(ProcessMonitor).where(ProcessMonitor.device_id == device_id))
    db.execute(delete(DiskMetric).where(DiskMetric.device_id == device_id))
    db.execute(delete(Metric).where(Metric.device_id == device_id))
    db.execute(delete(DeviceInventory).where(DeviceInventory.device_id == device_id))
    db.execute(delete(Device).where(Device.id == device_id))
    db.commit()
    return Response(status_code=204)

@app.patch('/api/devices/{device_id}/thresholds')
def thresholds(device_id: int, payload: ThresholdUpdate, db: DB, _: Annotated[User, Depends(current_user)]):
    if payload.warning_disk_percent >= payload.critical_disk_percent:
        raise HTTPException(400, 'Warning threshold must be below critical threshold')
    device = db.get(Device, device_id)
    if not device: raise HTTPException(404, 'Device not found')
    device.warning_disk_percent, device.critical_disk_percent = payload.warning_disk_percent, payload.critical_disk_percent
    db.commit()
    return {'ok': True}

@app.post('/api/devices/{device_id}/port-checks', status_code=201)
def add_port_check(device_id: int, payload: PortCheckCreate, db: DB, _: Annotated[User, Depends(current_user)]):
    if not db.get(Device, device_id): raise HTTPException(404, 'Device not found')
    check = PortCheck(device_id=device_id, **payload.model_dump())
    db.add(check); db.commit(); db.refresh(check)
    return {'id': check.id, 'name': check.name, 'host': check.host, 'port': check.port, 'protocol': check.protocol, 'timeout_seconds': check.timeout_seconds, 'udp_payload': check.udp_payload, 'expect_response': check.expect_response, 'enabled': check.enabled, 'latest': None}

@app.delete('/api/devices/{device_id}/port-checks/{check_id}', status_code=204)
def delete_port_check(device_id: int, check_id: int, db: DB, _: Annotated[User, Depends(current_user)]):
    check = db.scalar(select(PortCheck).where(PortCheck.id == check_id, PortCheck.device_id == device_id))
    if not check:
        raise HTTPException(404, 'Port check not found')

    # Remove historical results in the same transaction. This works immediately
    # even before/without the database FK migration above and avoids a 500 error.
    db.execute(delete(PortResult).where(PortResult.check_id == check.id))
    db.delete(check)
    db.commit()

@app.post('/api/devices/{device_id}/process-monitors', status_code=201)
def add_process_monitor(device_id: int, payload: ProcessMonitorCreate, db: DB, _: Annotated[User, Depends(current_user)]):
    if not db.get(Device, device_id):
        raise HTTPException(404, 'Device not found')
    existing = db.scalar(select(ProcessMonitor).where(ProcessMonitor.device_id == device_id, func.lower(ProcessMonitor.name) == payload.name.lower()))
    if existing:
        raise HTTPException(409, 'Process is already monitored')
    monitor = ProcessMonitor(device_id=device_id, name=payload.name)
    db.add(monitor)
    db.commit()
    db.refresh(monitor)
    running = db.scalar(select(func.count()).select_from(RunningProcess).where(RunningProcess.device_id == device_id, func.lower(RunningProcess.name) == payload.name.lower())) or 0
    return {'id': monitor.id, 'name': monitor.name, 'running': running > 0}

@app.delete('/api/devices/{device_id}/process-monitors/{monitor_id}', status_code=204)
def delete_process_monitor(device_id: int, monitor_id: int, db: DB, _: Annotated[User, Depends(current_user)]):
    monitor = db.scalar(select(ProcessMonitor).where(ProcessMonitor.id == monitor_id, ProcessMonitor.device_id == device_id))
    if not monitor:
        raise HTTPException(404, 'Process monitor not found')
    db.delete(monitor)
    db.commit()
    return Response(status_code=204)
