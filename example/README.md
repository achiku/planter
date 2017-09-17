
```
$ psql -U admin|root|whatever
> CREATE USER planter;
> REATE DATABASE planter OWNER planter;
```

```
$ psql -U planter -d planter
> \i ddl.sql
```

```
$ java -jar plantuml.jar -verbose example.uml
```
