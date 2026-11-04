FROM python:3.12-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY app.py .

ARG VCS_REPOSITORY_URL_FULL
ARG VCS_REF_HEAD_NAME
ARG VCS_REF_TYPE
ARG VCS_REF_HEAD_REVISION

ENV DD_GIT_REPOSITORY_URL=${VCS_REPOSITORY_URL_FULL}
ENV DD_GIT_COMMIT_SHA=${VCS_REF_HEAD_REVISION}

ENV VCS_REPOSITORY_URL_FULL=${VCS_REPOSITORY_URL_FULL}
ENV VCS_REF_HEAD_REVISION=${VCS_REF_HEAD_REVISION}
COPY --from=shared otel-env.sh /usr/local/bin/otel-env.sh

ENTRYPOINT ["/usr/local/bin/otel-env.sh"]
CMD ["python", "app.py"]
