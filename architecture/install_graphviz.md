# Installing Graphviz for Architecture Diagrams

To generate PNG images from the DOT files, you need Graphviz installed on your system.

## Installation Methods

### Windows

**Option 1: Download from Official Website**
1. Go to https://graphviz.org/download/
2. Download the Windows installer
3. Run the installer and follow the setup wizard
4. Add Graphviz to your PATH environment variable

**Option 2: Using Chocolatey (requires admin privileges)**
```cmd
choco install graphviz
```

**Option 3: Using Winget**
```cmd
winget install Graphviz.Graphviz
```

### macOS

**Using Homebrew:**
```bash
brew install graphviz
```

**Using MacPorts:**
```bash
port install graphviz
```

### Linux

**Ubuntu/Debian:**
```bash
sudo apt-get install graphviz
```

**CentOS/RHEL/Fedora:**
```bash
sudo yum install graphviz
# or
sudo dnf install graphviz
```

**Arch Linux:**
```bash
sudo pacman -S graphviz
```

## Generating the Diagram

Once Graphviz is installed, you can generate the architecture diagram:

```bash
cd architecture

# Generate PNG image from DOT file
dot -Tpng docker_monitor_architecture.dot -o docker_monitor_architecture.png

# Alternative formats
dot -Tsvg docker_monitor_architecture.dot -o docker_monitor_architecture.svg
dot -Tpdf docker_monitor_architecture.dot -o docker_monitor_architecture.pdf
```

## Using the Python Scripts

**For full diagrams library features:**
```bash
pip install -r requirements.txt
python generate_diagram.py
```

**For simple DOT file generation:**
```bash
python generate_diagram_simple.py
```

## Troubleshooting

### "command not found" error
- Ensure Graphviz is properly installed
- Check that the Graphviz bin directory is in your PATH
- On Windows, restart your command prompt after installation

### Permission errors
- On Windows, run command prompt as Administrator
- On Unix systems, ensure you have write permissions to the output directory

### Python import errors
```bash
pip install --upgrade diagrams graphviz
```

### Large diagrams
For complex diagrams that take long to render, try:
```bash
dot -Tpng -Gdpi=150 docker_monitor_architecture.dot -o docker_monitor_architecture.png
```