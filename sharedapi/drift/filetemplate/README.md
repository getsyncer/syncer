Given a git repository with a config file that looks like this:
```yaml
files:
  - source: "file1.txt"
    destination: "file1.txt"
```

When ran, this will find a config file inside the template repository named file1.txt and run the template engine
on it and output it to the destination file file1.txt inside the current repository.