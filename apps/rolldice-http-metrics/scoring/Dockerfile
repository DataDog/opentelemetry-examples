FROM python:3.9-alpine

WORKDIR /scoring

COPY . /scoring

RUN pip install -r requirements.txt

EXPOSE 5001

CMD ["python", "app.py"]
