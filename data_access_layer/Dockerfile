FROM python:3.8-slim-buster
COPY data_server.py /data_server.py
COPY leedsbeerquest.csv /leedsbeerquest.csv
RUN pip3 install flask
CMD /data_server.py
