package main

import (
	"fmt"
	"flag"
	"os"
)

func initf() {
 	for _, dir := range []string{".git", ".git/objects", ".git/refs"} {
 		if err := os.MkdirAll(dir, 0755); err != nil {
 			fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
 		}
 	}

 	headFileContents := []byte("ref: refs/heads/main\n")
 	if err := os.WriteFile(".git/HEAD", headFileContents, 0644); err != nil {
 		fmt.Fprintf(os.Stderr, "Error writing file: %s\n", err)
 	}

 	fmt.Println("Initialized git directory")
}

func config(name, email string) {
	data := fmt.Sprintf("name=%s\nemail=%s\n", name, email)
	err := os.WriteFile(".git/config", []byte(data), 0666)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing configuration: %s\n", err)
		os.Exit(1)
	}
}

// Usage: your_git.sh <command> <arg1> <arg2> ...
func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	
	 if len(os.Args) < 2 {
	 	fmt.Fprintf(os.Stderr, "usage:  mygit <command> [<args>...]\n")
	 	os.Exit(1)
	 }

	 catFileCmd := flag.NewFlagSet("cat-file", flag.ExitOnError)
	 pArg := catFileCmd.String("p", "", "file hash")

	 hashObjectCmd := flag.NewFlagSet("hash-object", flag.ExitOnError)
	 wArg := hashObjectCmd.String("w", "", "file name")

	 lsTreeCmd := flag.NewFlagSet("ls-tree", flag.ExitOnError)
	 nameBoolArg := lsTreeCmd.Bool("names-only", false, "read only names")
	 
	 _ = flag.NewFlagSet("write-tree", flag.ExitOnError)

	 commitTreeCmd := flag.NewFlagSet("commit-tree", flag.ExitOnError)
	 msgArg := commitTreeCmd.String("m", "", "commit message")
	 parArg := commitTreeCmd.String("p", "", "parrent commit hash")

	 configCmd := flag.NewFlagSet("config", flag.ExitOnError)
	 nameArg := configCmd.String("name", "", "author name")
	 emailArg := configCmd.String("email", "", "author email")

	 switch command := os.Args[1]; command {
	 case "init":
		 initf()

	 case "cat-file":
		 catFileCmd.Parse(os.Args[2:])
		 hash := *pArg
		 if hash == "" {
			 catFileCmd.Usage()
			 os.Exit(1)
		 }
		 catFile(hash)

	 case "hash-object":
		 hashObjectCmd.Parse(os.Args[2:])
		 filename := *wArg
		 if filename == "" {
			 hashObjectCmd.Usage()
			 os.Exit(1)
		 }
		 hashObject(filename)

	 case "ls-tree":
		 lsTreeCmd.Parse(os.Args[2:])
		 name := *nameBoolArg
		 hash := lsTreeCmd.Arg(0)
		 if lsTreeCmd.NArg() <= 0 {
			 lsTreeCmd.Usage()
			 os.Exit(1)
		 }
		 lsTree(hash, name)

	 case "write-tree":
		 writeTree(".")

	 case "commit-tree":
		 commitTreeCmd.Parse(os.Args[2:])
		 parent := *parArg
		 msg := *msgArg
		 hash := commitTreeCmd.Arg(0)
		 if commitTreeCmd.NArg() <= 0 {
			 commitTreeCmd.Usage()
			 os.Exit(1)
		 }
		 commitTree(hash, parent, msg)

	 case "config":
		 configCmd.Parse(os.Args[2:])
		 name := *nameArg
		 email := *emailArg
		 if name == "" || email == "" {
			 configCmd.Usage()
			 os.Exit(1)
		 }
		 config(name, email)

	 case "help":
		 fmt.Fprintf(
			 os.Stderr,
		 	"usage:  mygit <command> [<args>...]\n" +
		 	"\tinit						initialize git directory\n" +
			"\tcat-file -p <hash>				display blob file contents\n" +
			"\thash-object -w <filename>			write blob file object\n" +
			"\tls-tree [--names-only] <hash>			display tree object contents\n" +
			"\twrite-tree					write tree object\n" +
			"\tcommit-tree -p <parent> -m <message> <hash>	write tree commit object\n" +
			"\tconfig --name <name> --email <email>		configure git credentials\n",
		 )
		 os.Exit(0)
	
	 default:
	 	fmt.Fprintf(os.Stderr, "Unknown command %s\n", command)
	 	os.Exit(1)
	 }
}
