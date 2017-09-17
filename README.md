# Barely

Barely is a console [notmuch](http://notmuchmail.org/) frontend heavily inspired by [alot](http://github.com/pazz/alot),
which is awesome, but slow.
Therefore it tries to be fast and simple while staying very customizable.

## Features

Features include

- simplistic buffer interface
- searching for messages and threads.
- reading, replying and composing new messages
- sending and receiving attachments
- multiple accounts
- simple tab completion in the prompt

Things that are left to do

- encryption/signing support
- making the `:` prompt a bit friendlier (history)

## Installation

Barely is written in Go. To install it, get a working Go environment and type

```go get github.com/laochailan/barely```

If you don’t want Go to install stuff locally to your user directory, package
it properly for your distribution.

## Configuration

Barely looks for its config file in `~/.config/barely/config`. To get an
example file which contains all the standard settings run

```barely -config```

It is necessary to configure accounts in order to send mail.

## Usage

Barelys interface is arranged in buffers. A buffer is a view that can contain
search results or the content of a message for example.
In the bottom left you can see what buffer you are in and how many buffers are
open.

If you start notmuch, most of the time you will see a search buffer for unread
messages.

Unlike most MUAs Barely does not have mail directories but relies solely on
searches to organize your mail. This approach is much more flexible than
directories, especially when you have more than one mail account.

To do a search type `/`, a notmuch search term and then enter. Alternatively,
you can type `:search yoursearchterm`. To search for single messages instead of
threads type `:msearch searchterm`.

It is recommended to define keybindings for your favorite searches. I bind the
number keys to searches for my mail accounts for example.

```
[bindings]
key = 1 search to:account1@domain.com
key = 2 search to:account2@domain.com
key = 3 search to:account3@domain.com
key = 4 search to:account4@domain.com
key = 5 search to:account5@domain.com
```

Inside Barely you can always press the `?` key to get a list of possible
key bindings in that context.

For a full list, read the example config file.

If you are not familiar with notmuch, you might want read about its tagging
abilities to get the most out of Barely. I currently use

- offlineimap — to download my mail
- notmuch —  to index it
- barely — to read it and compose new messages
- msmtp — as sendmail command for barely
