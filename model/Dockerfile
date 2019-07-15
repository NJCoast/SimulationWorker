FROM 234514569215.dkr.ecr.us-east-1.amazonaws.com/model:runtime-v95

RUN apt update && apt install -y awscli jq

WORKDIR /app

ADD *.mat /app/
ADD execute.sh /app/

ADD run_ObtainingParametersCrossingPoint.sh /app/
ADD ObtainingParametersCrossingPoint /app/

ADD run_WebCentralAnalysis.sh /app/
ADD WebCentralAnalysis /app/