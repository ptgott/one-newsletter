# Divs to newsletters _!!_

## What is this?

It's a self-hosted tool for creating email newsletters for websites that lack them, or websites that _do_ send email newsletters but in a format that rubs you the wrong way, or peppered with way too many ads. You'll get a plain-looking list of links and captions, and you'll be free to save them to Pocket, print them out, or whatever else you do with email newsletters.

## How is it deployed?

The goal is to make this a lightweight binary that you can deploy to low-cost VMs (e.g., the least expensive [Digital Ocean VM](https://www.digitalocean.com/pricing/#standard-droplets) or t2.micro [spot instances](https://aws.amazon.com/ec2/spot/pricing/)). It should only be active once a day or so, make up to a few hundred HTTP requests, and parse a few KB of text.

It's also meant to be stateless, so you can build it into a machine image, use a self-healing VM service like AWS Auto Scaling, not need to worry about interruption. (We're assuming you're okay receiving your email newsletters a few minutes late every now and then.)

While it won't be designed for managed services like Google Cloud Run or AWS Lambda (mainly due to tales of [surprise DDoS-related bills and Google account lockouts](https://news.ycombinator.com/item?id=22027459)), it will be a single Go module that you can wrap with your Lambda function or deploy as a container to Cloud Run.
