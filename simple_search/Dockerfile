FROM python:slim-bookworm

RUN mkdir /app

COPY requirements.txt .
RUN pip install -r requirements.txt


WORKDIR /app

COPY . .

CMD [ "uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]