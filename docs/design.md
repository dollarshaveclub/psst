# psst - Platform Services Secret Tool 

| Title  | psst - Platform Services Secrets Tool |

| Date   | May 16, 2018 |

| Author(s) | Andrew Hamilton <andrew.hamilton@dollarshaveclub.com> |

| Reviewer(s) | |

| Approver(s) | |

| Revision Number | 2 |

| Status | Released |

## Executive Summary

psst is a secure tool for sharing secrets with other team members with little more than a connection to the VPN, a GitHub account in the DollarShaveClub organization and the psst tool.

## Goals

* Allow for users to securely share a secret with one or more people
* Allow any user to give a secret to another user
* Only allow the expected user(s) to see the given secret
* Require authentication to GitHub to allow for secret sharing
* No need to set up another service to share secrets. Every user should be added to the GitHub organization automatically.

## Non-goals

* Replace PGP for all use cases
* Provide long term storage for secrets for users
* Allow users to share secrets with services in different environments.

## Background

Currently sharing secrets at DSC is difficult and should be easier for all users to correctly share secrets. PGP/GPG are the currently the methods for sharing secrets. To use these tools, a user must create a private and public key pair and share that with other users. Currently sharing occurs in an ad hoc manner with the sender required to request the key from the user.  This can add some additional steps and difficulty when sending secrets so people will be more likely to share the secrets insecurely.

PGP/GPG is traditionally a difficult tool to use. Users must select an algorithm to use for the key as well as a key size. The user must securely store the private key and provide the public key through a trusted mechanism (key server, KeyBase.io, etc). Encryption and decryption are difficult actions to remember how to perform and usually require the user to lookup how to do it as it isn’t a common task.

## High-Level Design

psst is a command-line tool that will simplify secret sharing at DSC. psst will not do any encryption of the secrets but will utilize another service for this task. psst will use HashiCorp Vault for authorization, encryption and storage of secrets and GitHub for authentication. A user would be able to lookup a user based on a GitHub username or name. 

The Vault server used would be the production Vault server. Inside of Vault, a specific key space will be added in the form `/secret/psst/<username>/<secret>` that allows for writing by all users to any location but on the user with that username to read the secret.

## Detailed Design

psst is a command-line tool that will simplify secret sharing at DSC.

### Authentication

Authentication of a user will be handled with the GitHub Authentication system for HashiCorp Vault. The user will need to generate a GitHub token and place that on their system. This is generally kept in an environment variable name “GITHUB_TOKEN”. This token will be passed to vault for authentication and the user will received back an token specific to Vault for usage in writing secrets.

### Directory

We need a way to find a set of users and teams. We will use directories to find our set of users and teams. A directory will basically be a service that allows us to pull down a set of usernames and teams names and use them. Possible directories can include GitHub, AWS, Google Apps, etc.

GitHub is the original directory that we will utilize. It will be helpful to users attempting to share secrets to easily discover the GitHub username for the user. GitHub usernames are not always easy to figure out as some people use cryptic names. To make this easier the tool will take in a name from the user and search the organization for a username or account name that looks similar. If multiples are found, the user will be able to select the correct user.

There are times where a sensitive item needs to be shared with a team of users. psst will allow the user to lookup teams in GitHub as well and provide the secret to the group as a whole. By giving access to the group, we allow secrets to outlive all members of the team if needed. It is still wise to rotate secrets periodically or as team members join or leave a team.

### Storage

Storage is an encrypted medium that users in the directory above can access. The user should not need to do anything beyond a login to access the medium. Examples of storage include HashiCorp Vault, AWS Secret Store, etc.

Default storage will utilize HashiCorp Vault’s KV store (version 1). We will utilize the same cluster as production so that we can consider this service production level as well as alleviate extra operations overhead. This will securely hold secrets until a user is able to collect them.

The storage engine will properly authorize users based on Groups in the DollarShaveClub GitHub organization. The user will need to login to Vault with a proper GitHub token and be authenticated to be able to read and write secrets.

### Policies

The following policy will be added to Vault to allow all users to write to a secret to any user. This policy will be added to the “all” group so that it is inherited by new users.

``` hcl
path “/secret/psst/*” {
    capabilities = [“create”]
}
```

This policy will allow all users to create new entries.

The following policy will need to be added for every individual user in the “all” DollarShaveClub GitHub account and attached to that user in Vault:

``` hcl
path “/secret/psst/<username>/*” {
    capabilities = [“read”, “list”, “delete”]
}
```

This will need to be added as new users are added into GitHub. Unfortunately, there is not way for us to template these policies for all users unless we build a 3rd party service. This policy will prevent users from writing to their own space by default.

Since a member of a team might want to share secrets within their team, team members must be given write access to the drop location. We will need a different policy to allow for teams to have this ability.

``` hcl
path “/secret/psst/<team-name>/*” {
    capabilities = [“read”, “list”, “delete”]
}
```

psst will have a command to generate these policies automatically for new users but would require a manual run by someone with admin privileges (SRE, PlatformServices).

### Subcommands

psst will have the following subcommands available to the user:

* delete
* generate
* get
* list
* search
* share

The share command would be used to store secrets for another user. The user would provide a username or team name and a file from where to pull the secret data from. We would want to stay away from allowing secrets to be passed over the command line as that secret will be stored in the history file of the shell.

The share command will only take a key name that will be suffixed to the `/secret/psst/<username>/` key prefix to create the final path.

#### delete

The delete command will allow the user to delete an secret from the set stored in Vault.

#### generate

The generate command will generate a set of policies inside of the provided folder by the user. There is also an option for adding the new policies to Vault if the user has the proper permissions.

#### get

The get command would allow the user to collect the a secret from the Vault storage. By default it will print out the value of the secret to STDOUT. This command will take in a key name that is suffixed to a generated prefix. By default the would be `/secret/psst/<username>/`, but the user can also specify a team space to get the secret from instead.

#### list

The list command would allow the user to list all available secrets. List will list all keys underneath the user's drop as well as the drops that the user is currently listed as a member. It will specify which entity (user or team) that a secret currently belongs to.

#### search

The search command would allow the user to find a user or team of users that will receive the secret. The user and teams will be looked up in GitHub and returned to the user to use as part of the put command. If the user doesn’t enter a search term then all available users and teams will be presented.

## Alternatives Considered:

### PGP/GPG

#### Pros

* Strong encryption of secrets
* Widely used for this purpose

#### Cons

* Difficult to use for new users
* Difficult to share keys between users

### Jass

#### Pros

* Encryption using SSH keys in GitHub
* Easy for users to use
* Was used at Twitter when I was there

#### Cons

* Doesn’t work with SSH keys with Yubikeys
* Doesn’t work with GitHub organizations and groups for verification and searching

### GitHub GPG Keys

#### Pros

* Easy discovery of public keys based on local users
* Everyone at DSC engineering already has access to GitHub

#### Cons

* Difficult to build a tool use the GPG key since they don’t store the actual GPG but the raw cryptographic key

## Security Concerns

There are always security concerns around sharing secrets. There are layers of security around different aspects of the system.

The DollarShaveClub organization in GitHub currently requires two-factor authentication so it is less likely that a user’s account will be hacked.

The Vault server requires access to the VPN which also requires a two-factor code to authenticate with. The Vault server also requires authentication using a GitHub token. The Vault server also enforces access with policies that only trusted users have access to update. There is always the possibility for a trusted user to do bad things but we can help this by requiring policies to be added version control.

User laptops are an easier method for attacking this system. There is always the possibility that a user will note have a truly secured system but that will also only compromise that single user’s secrets. Based on the amount of trust given to this user, will determine the amount of access an attacker can gain. By use least permissions when providing access we can help to limit the number of issues we would have in case of a breach.

This template was an example in The Practice of Cloud System Administration: Designing and Operating Large Distributed Systems.
