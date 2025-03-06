import os
import subprocess, jinja2
import argparse

def compile_template(src: str, output: str, data={}):
    loader = jinja2.FileSystemLoader(searchpath=".")
    env = jinja2.Environment(loader=loader)
    template = env.get_template(src)

    outputText = template.render(data)
    with open(output, "w") as f:
        f.write(outputText)

def download_dataset(to, count=50):
    for i in range(count):
        subprocess.run(
            [
                "curl",
                "-sL",
                f"https://picsum.photos/1920/1080/?random={i}",
                "--output",
                f"{to}/{i}.jpg",
            ])

def get_paths(root_dir):
    output = []
    files = os.listdir(root_dir)
    for file in files:
        fp = os.path.join(root_dir, file)
        if os.path.isfile(fp):
            output.append(fp)
        else:
            output.extend(get_paths(fp))
    return output

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("-media-dir", type=str)
    parser.add_argument("-init-dataset", type=int, default=0)
    parser.add_argument("-prefix", type=str, default="")
    args = parser.parse_args()

    media_dir = os.path.abspath(args.media_dir)
    if not os.path.exists(media_dir):
        print("Media directory does not exist. Creating...")
        os.mkdir(media_dir)

    if args.init_dataset > 0:
        download_dataset(media_dir, count=args.init_dataset)
    
    pictures = get_paths(media_dir)
    if args.prefix:
        pictures = [
            {"key": picture.replace(media_dir + "/", args.prefix)}
            for picture in pictures
        ]
    
    dst = os.path.join(media_dir, "index.html")
    compile_template("template.html", dst, data={'pictures': pictures})
