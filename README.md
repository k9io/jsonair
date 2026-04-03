---
description: A cloud based, simple to use,  centralized configurations storage system.
---

# 1. What is JSONAir

**JSONAir** (  /jās-on-âr/ n. ) - is a simple-to-use, centralized cloud-based configuration storage system.

In many situations, legacy flat-file configuration systems cause headaches. For example, in containerized environments, having to manually edit a YAML file, get it into a container, and redeploy is a pain. In clustered environments, it might be even worse because the flat file needs to be distributed to each container.

Some software takes this into account by using system environment variables. While this improves the process, in most cases, it requires the containers to be restarted to re-read the new environment variable.

JSONAir tackles this by making configuration distribution ‘a service.’ Configurations for your software can easily be retrieved by calling a simple API.

JSONAir is agnostic to “how” configuration data is stored. To JSONAir, configuration data is just “data.” It might be legacy flat ASCII files, YAML, JSON, etc. JSONAir doesn’t care. However, this means that your software still needs to “validate” the configuration data.

While there are similar projects to JSONAir, we found them to be overly complicated for most of our use cases. The concept behind JSONAir is for it to remain as simple as possible. It is a configuration retrieval system, and it does not intend to validate, update, or modify configuration data. This makes the JSONAir API incredibly simple and uni-directional (read-only).

JSONAir is written in a memory-safe language (Golang) and can be used in containers and clusters itself for high availability. It is meant to be memory and CPU efficient.

Documenation can be found at: https://docs.k9.io/key9-identity/jsonair

