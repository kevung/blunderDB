import os
import shutil
import subprocess
import zipfile

CREATE_ARCHIVE = True
LANG = ['fr', 'en']


import os

def find_build_folders(root_folder):
    build_folders = []
    for dirpath, dirnames, filenames in os.walk(root_folder):
        for dirname in dirnames:
            if dirname == "build":
                build_folders.append(os.path.join(dirpath, dirname))
    return build_folders

def create_folder(folder_path):
    try:
        if os.path.exists(folder_path):
            shutil.rmtree(folder_path)
            print("Folder already exists. Overwriting...")
        os.makedirs(folder_path)
        print("Folder created successfully:", folder_path)
    except Exception as e:
            print("Error creating folder:", e)


def copy_directories_with_hierarchy(source_directories, destination_folder):
    try:
        for source_dir in source_directories:
            destination_dir = os.path.join(destination_folder, os.path.relpath(source_dir, os.path.commonpath(source_directories)))
            shutil.copytree(source_dir, destination_dir)
        print("Directories copied successfully with hierarchy.")
    except Exception as e:
        print("Error copying directories with hierarchy:", e)

def get_latest_commit():
    try:
        result = subprocess.run(['git', 'rev-parse', '--short', 'HEAD'], stdout=subprocess.PIPE)
        latest_commit = result.stdout.decode('utf-8').strip()
        return latest_commit
    except Exception as e:
        print("Error:", e)
        return None


def zip_folder(folder_path, zip_file_name):
    try:
        with zipfile.ZipFile(zip_file_name, 'w', zipfile.ZIP_DEFLATED) as zipf:
            for root, dirs, files in os.walk(folder_path):
                for file in files:
                    file_path = os.path.join(root, file)
                    relative_path = os.path.relpath(file_path, folder_path)
                    zipf.write(file_path, os.path.join(os.path.basename(folder_path), relative_path))
            print("Folder zipped successfully.")
    except Exception as e:
        print("Error zipping folder:", e)

def create_archive(root_dir, build_folders):
    #assumes all are build done and listed in build_folders
    build_dir = os.path.join(root_dir, 'build')
    latest_commit = get_latest_commit()
    for l in LANG:
        print("archive for lang:", l)
        project_name = 'blunderdb_'+l
        project_build_dir = os.path.join(build_dir, project_name)
        build_folders_lang = [os.path.join(s,l) for s in build_folders]

        copy_directories_with_hierarchy(build_folders_lang, project_build_dir)

        zip_filename = project_name +'_'  +latest_commit +'.zip'
        output_zip_file = os.path.join(build_dir, zip_filename)
        folder_to_zip = project_build_dir
        print("zip archive:", output_zip_file)
        zip_folder(folder_to_zip, output_zip_file)
        shutil.rmtree(project_build_dir)


def build_sphinx_docs(path, language):
    os.chdir(path)
    subprocess.run(["sphinx-build", "-b", "html", "-D",
        "language=" + language, "source", "build/" + language])
    print(f"Documentation built for {language} language in {path}")

def clean_previous_build(root_dir):
    build_folders = find_build_folders(root_dir)
    for folder in build_folders:
        if os.path.exists(folder):
            print(folder)
            shutil.rmtree(folder)

def main():
    root_dir = os.getcwd()
    build_dir = os.path.join(root_dir, "build")

    print("clean previous builds")
    clean_previous_build(root_dir)

    print("create build folder")
    create_folder(build_dir)

    build_sphinx_docs(root_dir, "fr")
    build_sphinx_docs(root_dir, "en")

    if CREATE_ARCHIVE:
        build_folders = find_build_folders(root_dir)
        print("Build folders found:")
        for folder in build_folders:
            print(folder)

        create_archive(root_dir, build_folders)

if __name__ == "__main__":
    main()
