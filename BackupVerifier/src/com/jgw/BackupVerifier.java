package com.jgw;

import java.io.FileInputStream;
import java.io.FileNotFoundException;
import java.io.IOException;
import java.util.Collections;
import java.util.Comparator;
import java.util.List;

import com.jgw.internal.BackupLoader;
import com.jgw.internal.Data;
import com.jgw.internal.DataBackup;
import com.jgw.internal.DataContext;

public class BackupVerifier {

	public static void main(String[] args) {

		try {
			FileInputStream fis = new FileInputStream(
					"s:\\Synchronized-Secure\\Backup-Verifier\\Backup-Verifier-Data.txt");

			DataContext dc = BackupLoader.parseFile(fis);

			System.out.println();
			System.out.println("-----------------------------------");

			List<Data> dataList = dc.getAllData();
			Collections.sort(dataList, new Comparator<Data>() {
				@Override
				public int compare(Data o1, Data o2) {
					return o1.getName().toLowerCase().compareTo(o2.getName().toLowerCase());
				}
			});

			for (Data d : dataList) {
				if (d.getName().equals("brain")) {
					continue;
				}

				OverviewPrinter.printData(d);
				System.out.flush();
				System.err.flush();
				BackupAnalyzer.analyzeData(d);
				System.out.flush();
				System.err.flush();
				GraphNodeRemover.analyze(d);
				System.out.flush();
				System.err.flush();

//				System.out.println(d.getName()+":");
//				for(DataBackup db : d.getDataBackups().values()) {
//					System.out.println(" db: "+db.getName());
//				}
//				
//				for(String str : d.getPasswordBackups()) {
//					System.out.println(" pdb: " + str);
//				}
//
				System.out.println();
			}

		} catch (FileNotFoundException e) {
			e.printStackTrace();
		} catch (IOException e) {
			e.printStackTrace();
		}

	}

}

class BackupAnalyzer {

	public static void analyzeData(Data d) {
		List<DataBackup> dbList = d.getDataBackups();

		if (dbList.size() == 1 && !d.isOneBackupOnly()) {
			System.err.println("* Not enough backups: " + d.getName());
		}

		boolean hasLocalBackup = false;
		boolean hasCloudBackup = false;

		for (DataBackup db : dbList) {
			if (db.getTag().equalsIgnoreCase("cloud")) {
				hasCloudBackup = true;
			} else {
				if (!db.getTag().equalsIgnoreCase(d.getTag())) {
					hasLocalBackup = true;
				} else {
					System.err.println("* Data backup has same tag as data: " + d.getName());
				}
			}
		}
		if (!hasLocalBackup && !d.isNoLocalNeeded()) {
			System.err.println("* Has no local backup: " + d.getName());
		}

		if (!hasCloudBackup && !d.isNoCloudNeeded()) {
			System.err.println("* Has no cloud backup: " + d.getName());
		}

		// Check the PDBs of the Data
		if (d.isEncrypted()) {
			checkPasswordBackupsOfData(d);
		}

		// next, check the PDBs of each of the Databackups
		checkPasswordBackupsOfDataBackups(d);

		// To analyze:
		// - number of backups (should be 2)
		// - one backup should be local, one in cloud (or two separate cloud)
		// o the local backup should not have the same tag as the data
		// o the passwords of the backups should be local, and in the cloud
		// o the local passwords backup should not have the same tags as the data
		// backup, or the data
		// - if the data has a password, it should be backed up
		//
		// - are we ok with brain password backups, generally speaking?
		// o of data?
		// o of data backups?
	}

	private static void checkPasswordBackupsOfDataBackups(Data d) {

		for (DataBackup db : d.getDataBackups()) {
			boolean hasCloudBackup = false;
			boolean hasLocalBackup = false;

			// Skip data backups which are not encrypted
			if (db.isUnencrypted()) {
				continue;
			}

			String localWarning = null;

			for (Data passwordBackup : db.getPasswordBackups()) {
				if (passwordBackup.getTag().equalsIgnoreCase("cloud")) {
					hasCloudBackup = true;
				} else {
					if (!passwordBackup.getTag().equalsIgnoreCase(db.getTag())
							&& !passwordBackup.getTag().equalsIgnoreCase(d.getTag())) {
						hasLocalBackup = true;
					} else {
						localWarning = "* Password data backup '" + passwordBackup.getName()
								+ "' has same tag as data backup or data itself: " + d.getName();
					}
				}

			}

			if (!hasLocalBackup) {
				if (localWarning != null) {
					System.err.println(localWarning);
				}
				System.err.println(
						"* Password backup of data backup '" + db.getName() + "' has no local backup: " + d.getName());
			}

			if (!hasCloudBackup) {
				System.err.println(
						"* Password backup of data backup '" + db.getName() + "' has no cloud backup: " + d.getName());
			}

		}

	}

	private static void checkPasswordBackupsOfData(Data d) {
		boolean hasCloudBackup = false;
		boolean hasLocalBackup = false;

		String localWarningMessage = null;

		for (Data passwordBackup : d.getPasswordBackups()) {
			if (passwordBackup.getTag().equalsIgnoreCase("cloud")) {
				hasCloudBackup = true;
			} else {
				if (!passwordBackup.getTag().equalsIgnoreCase(d.getTag())) {
					hasLocalBackup = true;
				} else {
					localWarningMessage = "* Password data backup has same tag as data: " + d.getName();
				}
			}
		}

		if (!hasLocalBackup) {
			if (localWarningMessage != null) {
				System.err.println(localWarningMessage);
			}
			System.err.println("* Password data backup of data has no local backup: " + d.getName());
		}

		if (!hasCloudBackup) {
			System.err.println("* Password data backup of data has no cloud backup: " + d.getName());
		}
	}
}

class OverviewPrinter {

	public static void printData(Data d) {

		System.out.println(d.getName() + (d.getTag() != null ? " [" + d.getTag() + "]" : "") + ":");
//		System.out.println();
		for (DataBackup db : d.getDataBackups()) {
			printDataBackup(db, 1);
		}

		for (Data str : d.getPasswordBackups()) {
			printPasswordDataBackup(str, 1);
		}

	}

	public static void printDataBackup(DataBackup db, int tabs) {
		System.out.println(tab(tabs) + "- db: " + db.getName() + (db.getTag() != null ? " [" + db.getTag() + "]" : ""));
		for (Data pdb : db.getPasswordBackups()) {
			printPasswordDataBackup(pdb, tabs + 5);
		}

	}

	public static void printPasswordDataBackup(Data passwordDataBackup, int tabs) {
		System.out.println(tab(tabs) + (tabs == 1 ? "- " : "") + "pdb: " + passwordDataBackup.getName()
				+ (passwordDataBackup.getTag() != null ? " [" + passwordDataBackup.getTag() + "]" : ""));
	}

	private static String tab(int c) {
		return (c > 0 ? " " + tab(c - 1) : "");
	}

}