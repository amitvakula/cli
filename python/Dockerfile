FROM python:3.6-alpine3.7

RUN apk add --no-cache postgresql
RUN pip install --upgrade pip setuptools wheel

# Pre-build psycopg2
RUN apk add --no-cache --virtual .build-deps gcc python3-dev musl-dev postgresql-dev \
	&& pip install psycopg2>=2.7.6 \
	&& apk del --no-cache .build-deps

RUN mkdir -p /db/ \
	&& chown -R postgres:postgres /db/ \
	&& su postgres -c "pg_ctl initdb --pgdata=/db/postgres"

# Print all queries
RUN echo "log_statement = 'all'" >> /db/postgres/postgresql.conf

RUN mkdir -p /run/postgresql \
	&& chown -R postgres:postgres /run/postgresql/

COPY . /app/src
RUN pip install -r /app/src/test-requirements.txt
RUN pip install -e /app/src

WORKDIR /app/src

