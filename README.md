# Divs to newsletters _!!_

## What is this?

It's a self-hosted tool for creating email newsletters for websites that lack them, or websites that _do_ send email newsletters but in a format that rubs you the wrong way, or peppered with way too many ads. You'll get a plain-looking list of links and captions, and you'll be free to save them to Pocket, print them out, or whatever else you do with email newsletters.

## How is it deployed? Some possibilities:

#### Hosted services

- While Google Cloud Run would make be basically free for this sort of thing, the possibility of [surprise DDoS-related bills and Google account lockouts](https://news.ycombinator.com/item?id=22027459) seemed like it wasn't worth it. We could consider adding a Terraform module for this in the future, though.

#### VMs

These are low-cost, allow for more flexibility, and don't infinitely and accidentally scale up, but may not be the most appropriate for this workload, which isn't designed to run constantly.

- [Digital Ocean VMs](https://www.digitalocean.com/pricing/#standard-droplets): \$5 per month for the smallest instance type

- [EC2 spot instances](https://aws.amazon.com/ec2/spot/pricing/): Around \$2.50 per month for a t2.micro.

- Whatever the user is already using! This won't be a heavyweight workload, so it can basically sit anywhere. The Terraform module can expose inputs for SSH keys, then upload the application binary to the remote machine and run it. Or skip the Terraform module and just provide the binary.
