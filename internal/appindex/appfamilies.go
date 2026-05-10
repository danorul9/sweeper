package appindex

import (
	"path/filepath"
	"strings"
)

var AppFamilies = []AppFamily{
	{
		Vendor: "Google",
		BundleIDs: []string{
			"com.google.drive",
			"com.google.Chrome",
			"com.google.BackupAndSync",
			"com.google.earth",
			"com.google.desktop",
			"com.google.photos",
		},
		Folders: []string{
			"Google",
			"Google Drive",
			"com.google.drive",
			"Google/Chrome",
			"com.google.Chrome",
			"com.google.earth",
			"com.google.BackupAndSync",
		},
	},
	{
		Vendor: "Adobe",
		BundleIDs: []string{
			"com.adobe.Photoshop",
			"com.adobe.Illustrator",
			"com.adobe.InDesign",
			"com.adobe.PremierePro",
			"com.adobe.AfterEffects",
			"com.adobe.Lightroom",
			"com.adobe.Acrobat",
			"com.adobe.CreativeCloud",
		},
		Folders: []string{
			"Adobe",
			"Adobe Creative Cloud",
			"com.adobe.*",
			"AdobeGCCClient",
		},
	},
	{
		Vendor: "JetBrains",
		BundleIDs: []string{
			"com.jetbrains.intellij",
			"com.jetbrains.goland",
			"com.jetbrains.WebStorm",
			"com.jetbrains.PyCharm",
			"com.jetbrains.CLion",
			"com.jetbrains.PhpStorm",
			"com.jetbrains.RubyMine",
		},
		Folders: []string{
			"JetBrains",
			"com.jetbrains.*",
			"JetBrains Toolbox",
		},
	},
	{
		Vendor: "Microsoft",
		BundleIDs: []string{
			"com.microsoft.VSCode",
			"com.microsoft.Excel",
			"com.microsoft.Word",
			"com.microsoft.PowerPoint",
			"com.microsoft.Outlook",
			"com.microsoft.Teams",
			"com.microsoft.Edge",
		},
		Folders: []string{
			"Microsoft",
			"com.microsoft.*",
			"Microsoft Edge",
			"Microsoft_Office",
		},
	},
	{
		Vendor: "Apple",
		BundleIDs: []string{
			"com.apple.*",
		},
		Folders: []string{
			"com.apple.*",
		},
	},
}

func hasGlob(s string) bool {
	return strings.ContainsAny(s, "*?[")
}

func (idx *AppIndex) IsFamilyInstalled(family AppFamily) bool {
	if family.Vendor == "Apple" {
		return true
	}
	for _, bid := range family.BundleIDs {
		if hasGlob(bid) {
			for existing := range idx.BundleIDs {
				if matched, _ := filepath.Match(bid, existing); matched {
					return true
				}
			}
		} else {
			if _, ok := idx.BundleIDs[bid]; ok {
				return true
			}
		}
	}
	return false
}

func (idx *AppIndex) FamilyForFolder(folderName string) *AppFamily {
	for _, family := range AppFamilies {
		for _, f := range family.Folders {
			if f == folderName {
				return &family
			}
			if hasGlob(f) {
				if matched, _ := filepath.Match(f, folderName); matched {
					return &family
				}
			}
		}
	}
	return nil
}
