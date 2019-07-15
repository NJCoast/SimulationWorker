FROM debian:latest

RUN apt update && apt install -y libncurses5 libxext6 libxt6 && mkdir -p /opt/matlab/runtime
ADD Runtime/ /opt/matlab/runtime/