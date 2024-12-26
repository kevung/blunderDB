# Configuration file for the Sphinx documentation builder.
#
# For the full list of built-in configuration values, see the documentation:
# https://www.sphinx-doc.org/en/master/usage/configuration.html

# -- Project information -----------------------------------------------------
# https://www.sphinx-doc.org/en/master/usage/configuration.html#project-information

project = 'blunderDB'
copyright = '2024, Kevin UNGER <blunderdb@proton.me>'
author = 'Kevin UNGER <blunderdb@proton.me>'
release = '0.1.0'

# -- General configuration ---------------------------------------------------
# https://www.sphinx-doc.org/en/master/usage/configuration.html#general-configuration

extensions = [
        'sphinx_rtd_theme',
        ]

templates_path = ['_templates']
locale_dirs = ['locale/']
gettext_compact = False
html_favicon = 'img/favicon.jpg'
exclude_patterns = []

language = 'fr'

# -- Options for HTML output -------------------------------------------------
# https://www.sphinx-doc.org/en/master/usage/configuration.html#options-for-html-output

html_theme = "sphinx_rtd_theme"
html_static_path = ['_static']
html_show_sphinx = False
html_show_sourcelink = False

html_context = {
        'languages': [["en", "../en"], ["fr", "../fr"]]
        }
