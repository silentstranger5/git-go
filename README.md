[![progress-banner](https://backend.codecrafters.io/progress/git/ca13f8f0-59f2-4642-a850-c04851072bf3)](https://app.codecrafters.io/users/codecrafters-bot?r=2qF)

This is a starting point for Go solutions to the
["Build Your Own Git" Challenge](https://codecrafters.io/challenges/git).

In this challenge, you'll build a small Git implementation that's capable of
initializing a repository, creating commits and cloning a public repository.
Along the way we'll learn about the `.git` directory, Git objects (blobs,
commits, trees etc.), Git's transfer protocols and more.

**Note**: If you're viewing this repo on GitHub, head over to
[codecrafters.io](https://codecrafters.io) to try the challenge.

# Passing the first stage

The entry point for your Git implementation is in `cmd/mygit/main.go`. Study and
uncomment the relevant code, and push your changes to pass the first stage:

```sh
git add .
git commit -m "pass 1st stage" # any msg
git push origin master
```

That's all!

# Stage 2 & beyond

Note: This section is for stages 2 and beyond.

1. Ensure you have `go` installed locally
1. Run `./your_git.sh` to run your Git implementation, which is implemented in
   `cmd/mygit/main.go`.
1. Commit your changes and run `git push origin master` to submit your solution
   to CodeCrafters. Test output will be streamed to your terminal.

# Testing locally

The `your_git.sh` script is expected to operate on the `.git` folder inside the
current working directory. If you're running this inside the root of this
repository, you might end up accidentally damaging your repository's `.git`
folder.

We suggest executing `your_git.sh` in a different folder when testing locally.
For example:

```sh
mkdir -p /tmp/testing && cd /tmp/testing
/path/to/your/repo/your_git.sh init
```

To make this easier to type out, you could add a
[shell alias](https://shapeshed.com/unix-alias/):

```sh
alias mygit=/path/to/your/repo/your_git.sh

mkdir -p /tmp/testing && cd /tmp/testing
mygit init
```

# Addendum
This repository contains a complete implementation of [Git](https://en.wikipedia.org/wiki/Git) in Golang. It is written using [Codecrafters](https://codecrafters.io/) tutorial. It implements all core features: work with blobs, trees, and commits. Source files are located at `cmd/mygit`.

| File Name | File Description |
|-----------|------------------|
| file.go   | Implements work with file blobs. See [Object storage](https://en.wikipedia.org/wiki/Object_storage) |
| main.go   | Implements git directory initialization and CLI processing |
| tree.go   | Implements work with tree objects and commits. |