# Introduction

Cannon is a browser based file previewer for terminal file managers like [lf](https://github.com/gokcehan/lf).

It follows the rules defined in its configuration to sample and convert each selected file into its web equivalent and then serves the converted file to the localhost browser from an internal web server. It was originally written for Windows 8 but should run properly on any platform supported by Go.

# Installation

Installing Cannon requires a recent version of [Go](https://go.dev/).

```
go install -v github.com/ccammack/cannon@e5df1174d1838baccc984382d11a53a644ce4b6d
```

After installation, copy the default [configuration file](https://github.com/ccammack/cannon/blob/main/cannon.yml) to the appropriate location.

* On Windows, the configuration file should be copied to:
  * C:\Users\\**USERNAME**\\AppData\\Local\\cannon\\cannon.yml

* On Linux and other systems, the configuration file should be copied to:
  * ~/.config/cannon/cannon.yml

Cannon depends on external programs for MIME detection, which are configured in the `mime:` section of `cannon.yml`. By default, this is accomplished on Linux using the built-in `file` command and on Windows using the version of `file` that ships with [`git` for Windows](https://gitforwindows.org/). If needed, install [`git` for Windows](https://gitforwindows.org/) and modify `cannon.yml` to use the correct path.

```
mime:
  default: ['file', '-b', '--mime-type', '{input}']
  windows: ['C:\Program Files\Git\usr\bin\file', '-b', '--mime-type', '{input}']
```

# Running

Starting `cannon` will automatically open a web browser for display. This defaults to [Chrome](https://www.google.com/chrome/) but can be configured in the `browser:` section of `config.yml` as needed. Browsers usually disable autoplay by default, so set the appropriate option to re-enable it in your browser for faster previews.

```
browser:
  default: ['google-chrome', '--autoplay-policy=no-user-gesture-required', '{url}']
  windows: ['C:\Program Files (x86)\Google\Chrome\Application\chrome.exe', '--autoplay-policy=no-user-gesture-required', '{url}']
```

Run `cannon --start` in one console to start the server and open the browser, then open a second console and give it a native HTML media file to display, such as [Rube Goldberg's Self-Operating Napkin (1931)](Self-Operating_Napkin.gif "Image source: Wikimedia Commons").

```
cannon --start
```

```
cannon Self-Operating_Napkin.gif
cannon --stop
```

# Integration

Configuring [lf](https://github.com/gokcehan/lf) to use Cannon requires that one map a key to toggle the server on and off and set the previewer. Integrating other file managers should follow a similar pattern.

On Windows, change the `lf` configuration file `C:\Users\USERNAME\AppData\Local\lf\lfrc` to `map` the `T` key to toggle the server and set the `previewer` command to Cannon.

```
# configure Cannon
map T &cannon --toggle
set previewer cannon
```

On Linux, change the `lf` configuration file `~/.config/lf/lfrc` in a similar fashion.

```
# configure Cannon
map T $(cannon --toggle &)
set previewer cannon
```

Start `lf` as usual and then press `T` to start the server and open the preview browser.
Browse the file system using `lf` and file previews should appear in the browser window.
Press `T` again to stop the server.

![Cannon preview](cannon-preview.png "Cannon preview")

# File Conversion Rules

> Each of the command-related sections in the configuration file allows one to specify a `default` command and then override that with a *platform-specific* command
using the names defined in the **$GOOS** list [here](https://go.dev/doc/install/source#environment).

Web browsers support a limited set of native HTML media files, so displaying other file types requires installing additional software and configuring Cannon to handle each file type. For example, [`ffmpeg`](https://ffmpeg.org/) can convert most audio and video files to a native format, [`ImageMagick`](https://imagemagick.org/) can convert most image formats, and [`MuPDF`](https://mupdf.com/) can convert the first page of a PDF to an image so it can be displayed in the browser.

To do this, install the desired conversion software, then configure the `file_conversion_rules:` section in `cannon.yml` to manage each conversion. The `file_conversion_rules:` section is processed in order from top to bottom. Each rule attempts to match the file against a list of file extensions (`ext`) and MIME types (`mime`).

When a match is found, Cannon will run the associated `command` to produce an output file and then serve the file using the specified HTML `tag`. If a rule does not specify a `command`, Cannon will just serve the original file.

For example, `mp3` and `wav` files can be served directly using the `<audio>` tag without running a conversion. The `{url}` parameter is required for each `tag` definition.

```
- # native html5 audio formats do not need conversion
  ext: [mp3, wav]
  tag: <audio autoplay loop controls src='{url}'>
```

All other audio files require sampling and conversion using `ffmpeg` to create a short audio preview. The `{input}` and `{output}` parameters are required for this conversion. The `{output}` parameter may specify an extension.

```
- # use ffmpeg to sample the first few seconds of non-native audio files
  mime: [audio]
  command:
    default: ['ffmpeg', '-ss', '0', '-i', '{input}', '-t', '5', '{output}.wav']
    windows: ['ffmpeg', '-ss', '0', '-i', '{input}', '-t', '5', '{output}.wav']
  tag: <audio autoplay loop controls src='{url}'>
```

If none of the conversion rules match, Cannon will display the first 4K bytes of the file.
If a conversion `command` is not provided, Cannon will serve the input file.
If the conversion `command` does not specify an `{output}` parameter or gives an error when run,
Cannon will serve the combined `stdout+stderr` created by the conversion `command`.
If a `tag` parameter is not provided, Cannon will display the output inside `<xmp>` tags.
