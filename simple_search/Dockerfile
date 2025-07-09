FROM python:slim-bookworm

RUN mkdir /app

COPY requirements.txt .
RUN pip install -r requirements.txt


WORKDIR /app

COPY . .

EXPOSE 8000
CMD [ "uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]