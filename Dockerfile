FROM ubuntu:trusty
COPY bin/linux_amd64/frontend /trees/bin/
COPY static/ /trees/static/
COPY data/trees.json /trees/data/
EXPOSE 80
ENTRYPOINT ["/trees/bin/frontend"]
CMD ["--addr=:80", "--data=/trees/data/trees.json", "--static=/trees/static"]
