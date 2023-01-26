# Introduction

Cannon is a brute-force browser-based file previewer for terminal file managers like [lf](https://github.com/gokcehan/lf).

Its primary advantage over options like [Pistol](https://github.com/doronbehar/pistol) is that it runs on older versions of Windows. It also runs on Linux and should work elsewhere if properly configured.

Cannon follows the rules defined in its configuration to sample and convert each selected file into its web equivalent and then serves the converted file to the browser from an internal web server.

![Rube Goldberg's Self-Operating Napkin](Rube_Goldberg's__Self-Operating_Napkin__(cropped).gif "Image source: Wikimedia Commons")

# Installation

Downloading and installing Cannon requires both [git](https://git-scm.com/) and [go](https://go.dev/). On Windows, the easiest way to install them might be to use [Chocolatey](https://chocolatey.org/) or another package manager.

Assuming that you have [git](https://community.chocolatey.org/packages?q=git) and [go](https://community.chocolatey.org/packages?q=go) installed, download and build Cannon.

	git https://github.com/ccammack/cannon.git
	cd cannon
	go install .

Next, copy the configuration file (cannon.yaml) into the proper [XDG-specified](https://github.com/adrg/xdg) location:

* On Windows, the configuration file should be copied to one of these locations:
  * C:\\Users\\USERNAME\\.config\\cannon\\cannon.yaml
  * C:\Users\\USERNAME\\AppData\\Local\\cannon\\cannon.yaml

* On Linux and other systems, the configuration file should be copied to:
  * ~/.config/cannon/cannon.yaml

The default configuration also relies on [muPDF](https://community.chocolatey.org/packages?q=mupdf) and [ffmpeg](https://community.chocolatey.org/packages?q=ffmpeg) for file conversion, so install them too.

By default, MIME detection is accomplished on Linux using the built-in *file* command and on Windows using the version of *file* that ships with *git* for Windows.

# Integration

Configuring [lf](https://github.com/gokcehan/lf) to use Cannon requires that one set the previewer and also map a key to toggle the server on and off.

In my case, I changed the configuration file *C:\\Users\\ccammack\\AppData\\Local\\lf\\lfrc* to set the *previewer* to Cannon and *map* the **T** key to start and stop the server.
Specifying full paths usually works better on Windows.

	# map the [T] key to toggle the preview server and set the custom file previewer
	map T push &C:/Users/ccammack/go/bin/cannon.exe<space>--toggle<enter>
	set previewer "C:/Users/ccammack/go/bin/cannon.exe"

Integrating other file managers should follow a similar pattern.

# Running

Start *lf* as usual and then press **T** to open the preview browser. Browse the file system using *lf* and file previews should appear in the browser window.

# Configuration

Each of the command-related sections in the configuration file allows one to specify a *default* command and then override that with a *platform-specific* command
using the names defined in the **$GOOS** list [here](https://go.dev/doc/install/source#environment).

Change the default server port from 8888 to another value if needed:

	# specify server address and port (select an unused port above 1023)
	port: 8888

Change the default browser from Chrome to your browser of choice if needed.
Browsers usually disable autoplay by default, so set the appropriate option to re-enable it in your browser for faster previews.
The *{url}* parameter is required.

	browser:
		default: ['chrome', '--autoplay-policy=no-user-gesture-required', '{url}']
		windows: ['C:\Program Files (x86)\Google\Chrome\Application\chrome.exe', '--autoplay-policy=no-user-gesture-required', '{url}']

Specify the *file* command for your platform to perform MIME type detection:
The *{file}* parameter is required.

	mime:
		default: ['file', '-b', '--mime-type', '{file}']
		windows: ['C:\Program Files\Git\usr\bin\file', '-b', '--mime-type', '{file}']

*lf* requires a non-zero exit code to prevent it from trying to cache the preview. Other file managers may have similar requirements.
Specify the exit code you want Cannon to set for all preview calls as needed.

	# cannon will set this exit value at the command line: $ cannon <file>
	#     lf requires a non-zero return value to prevent it from trying to cache the preview
	exit: 255

The *file_conversion_rules* in the configuration are processed in order from top to bottom.
Each rule attempts to match the file against a list of file extensions and MIME types.

When a match is found, Cannon will run the associated *command* to produce an output file and then serve the file using the specified HTML *tag*.
If a rule does not specify a command, Cannon will just serve a temporary copy of the original file.

For example, *mp3* and *wav* files can be served directly using the *audio tag* without running a conversion command.
The *{src}* parameter is required for each *tag* definition.

	- # native html5 audio formats do not need conversion
		ext: [mp3, wav]
		tag: <audio autoplay loop controls src='{src}'>

All other audio files require sampling and conversion using *ffmpeg* to create a short audio preview.
The *{input}* and *{output}* parameters are required for each conversion command.
The *{output}* parameter may specify an extension.

	- # use ffmpeg to sample the first few seconds of non-native audio files
		mime: [audio]
		command:
			default: ['ffmpeg', '-ss', '0', '-i', '{input}', '-t', '5', '{output}.wav']
			windows: ['ffmpeg', '-ss', '0', '-i', '{input}', '-t', '5', '{output}.wav']
		tag: <audio autoplay loop controls src='{src}'>

If none of the conversion rules match, Cannon will display the first 4K bytes of the file.
