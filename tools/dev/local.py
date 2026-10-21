"""Workspace-local UniGate development tools. Secrets stay in ignored .local/runtime.json."""
from pathlib import Path
import argparse
import base64
import hashlib
import io
import json
import os
import secrets
import shutil
import subprocess
import sys
import tarfile
import urllib.request
import zipfile

ROOT = Path(__file__).resolve().parents[2]
LOCAL = ROOT / '.local'
PG_VERSION = '18.6.0'
PG_SHA256 = 'b7ef2d03588e0439f0d0a2250f43905f7464f1e96ace47703ceaf79094198150'


def config():
    LOCAL.mkdir(exist_ok=True)
    file = LOCAL / 'runtime.json'
    if not file.exists():
        file.write_text(json.dumps({
            'databasePassword': secrets.token_urlsafe(30),
            'masterKey': base64.b64encode(secrets.token_bytes(32)).decode(),
            'bootstrapUsername': 'admin', 'bootstrapPassword': secrets.token_urlsafe(24), 'port': 5432,
        }, indent=2) + '\n', encoding='utf-8')
    return json.loads(file.read_text(encoding='utf-8'))


def environment(test=False):
    c = config()
    env = os.environ.copy()
    env.update({
        'UNIGATE_DB_URL': f'jdbc:postgresql://127.0.0.1:{c["port"]}/unigate' + ('_test' if test else ''),
        'UNIGATE_DB_USER': 'unigate', 'UNIGATE_DB_PASSWORD': c['databasePassword'],
        'UNIGATE_MASTER_KEY': c['masterKey'], 'UNIGATE_BOOTSTRAP_USERNAME': c['bootstrapUsername'],
        'UNIGATE_BOOTSTRAP_PASSWORD': c['bootstrapPassword'],
        'UNIGATE_COOKIE_SECURE': 'false', 'UNIGATE_ALLOWED_ORIGIN': 'http://localhost:5173',
        'UNIGATE_KEY_DIRECTORY': str(LOCAL / ('test-keys' if test else 'keys')),
        'UNIGATE_STORAGE_DIRECTORY': str(LOCAL / ('test-artifacts' if test else 'artifacts')),
    })
    if test:
        env.update({
            'UNIGATE_TEST_DB_URL': env['UNIGATE_DB_URL'],
            'UNIGATE_TEST_DB_USER': env['UNIGATE_DB_USER'],
            'UNIGATE_TEST_DB_PASSWORD': env['UNIGATE_DB_PASSWORD'],
        })
    return env


def run(args, *, cwd=ROOT, env=None):
    # Only fixed tool commands/arguments are passed; passwords are never shell text.
    return subprocess.run(args, cwd=cwd, env=env, shell=os.name == 'nt', check=True)


def pg_tools():
    if os.name != 'nt':
        raise SystemExit('Use Compose or your PostgreSQL installation on this platform.')
    directory = LOCAL / 'postgres-18.6'
    if not (directory / 'bin/initdb.exe').exists():
        downloads = LOCAL / 'downloads'
        downloads.mkdir(parents=True, exist_ok=True)
        name = f'embedded-postgres-binaries-windows-amd64-{PG_VERSION}.jar'
        jar = downloads / name
        if not jar.exists():
            url = f'https://repo.maven.apache.org/maven2/io/zonky/test/postgres/embedded-postgres-binaries-windows-amd64/{PG_VERSION}/{name}'
            with urllib.request.urlopen(url, timeout=90) as source, jar.open('wb') as target:
                shutil.copyfileobj(source, target)
        if hashlib.sha256(jar.read_bytes()).hexdigest() != PG_SHA256:
            raise SystemExit('PostgreSQL artifact SHA256 mismatch; refusing to execute.')
        directory.mkdir(exist_ok=True)
        with zipfile.ZipFile(jar) as archive:
            with tarfile.open(fileobj=io.BytesIO(archive.read('postgres-windows-x86_64.txz')), mode='r:xz') as tar:
                for member in tar.getmembers():
                    if not (directory / member.name).resolve().is_relative_to(directory.resolve()):
                        raise SystemExit('Unsafe PostgreSQL archive member.')
                tar.extractall(directory, filter='data')
    return directory / 'bin'


def db_start():
    c = config()
    binaries = pg_tools()
    data = LOCAL / 'pgdata'
    if not (data / 'PG_VERSION').exists():
        password_file = LOCAL / 'initdb-password'
        password_file.write_text(c['databasePassword'], encoding='utf-8')
        try:
            subprocess.run([str(binaries / 'initdb.exe'), '-D', str(data), '-U', 'unigate',
                            '--pwfile=' + str(password_file), '-A', 'scram-sha-256', '--encoding=UTF8', '--locale=C'], check=True)
        finally:
            password_file.unlink(missing_ok=True)
    status = subprocess.run([str(binaries / 'pg_ctl.exe'), '-D', str(data), 'status'],
                            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    if status.returncode:
        # DEVNULL prevents the detached Windows server from keeping an inherited output pipe open.
        subprocess.run([str(binaries / 'pg_ctl.exe'), '-D', str(data), '-l', str(LOCAL / 'postgres.log'),
                        '-o', f'-h 127.0.0.1 -p {c["port"]}', '-w', 'start'],
                       stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=True)
    modules = LOCAL / 'python'
    sys.path.insert(0, str(modules))
    try:
        import psycopg
    except ImportError:
        run([sys.executable, '-m', 'pip', 'install', '--target', str(modules), 'psycopg[binary]==3.3.6', '--disable-pip-version-check'])
        import psycopg
    with psycopg.connect(host='127.0.0.1', port=c['port'], user='unigate', password=c['databasePassword'],
                          dbname='postgres', autocommit=True) as connection:
        names = {row[0] for row in connection.execute('select datname from pg_database')}
        for name in ['unigate', 'unigate_test']:
            if name not in names:
                connection.execute(psycopg.sql.SQL('CREATE DATABASE {}').format(psycopg.sql.Identifier(name)))
    print(f'PostgreSQL ready on 127.0.0.1:{c["port"]}; development and test databases available.')
    print('Initial account credentials are stored only in .local/runtime.json. First login requires a password change.')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=['db-start', 'db-stop', 'backend', 'frontend', 'verify'])
    args = parser.parse_args()
    os.chdir(ROOT)
    if args.command == 'db-start':
        db_start()
    elif args.command == 'db-stop':
        subprocess.run([str(pg_tools() / 'pg_ctl.exe'), '-D', str(LOCAL / 'pgdata'), '-m', 'fast', '-w', 'stop'], check=True)
    elif args.command == 'backend':
        run([shutil.which('mvn') or 'mvn', '-f', 'backend/pom.xml', 'spring-boot:run'], env=environment())
    elif args.command == 'frontend':
        run([shutil.which('npm') or 'npm', '--prefix', 'frontend', 'run', 'dev'])
    elif args.command == 'verify':
        run([shutil.which('mvn') or 'mvn', '-f', 'backend/pom.xml', 'verify'], env=environment(test=True))
        run([shutil.which('npm') or 'npm', '--prefix', 'frontend', 'run', 'build'])
        run([shutil.which('npm') or 'npm', '--prefix', 'frontend', 'test'])


if __name__ == '__main__':
    main()
