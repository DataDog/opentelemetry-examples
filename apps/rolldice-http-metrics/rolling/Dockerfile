FROM python:3.9-alpine

WORKDIR /rolling

COPY . /rolling

RUN pip install -r requirements.txt

EXPOSE 5004

CMD ["python", "app.py"]
