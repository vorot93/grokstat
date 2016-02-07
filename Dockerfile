FROM scratch
MAINTAINER Artem Vorotnikov <artem@vorotnikov.me>
COPY ./bin/grokstat /usr/bin/grokstat

ENTRYPOINT [ "/usr/bin/grokstat" ]
