package com.jgw.internal;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.util.ArrayList;
import java.util.List;

public class BackupLoader {
	private static enum CurrentMode {
		INIT, DATA, DATA_BACKUP, PASSWORD_BACKUP
	};

	public static DataContext parseFile(InputStream is) throws IOException {
		CurrentMode mode = CurrentMode.INIT;

		DataContext context = new DataContext();

		BufferedReader br = new BufferedReader(new InputStreamReader(is));

		boolean startSeen = false;
		boolean endSeen = false;

		String str;

		while (null != (str = br.readLine())) {

			str = str.trim();
			if (str.length() == 0) {
				continue;
			}
			if (endSeen) {
				continue;
			}

			if (str.equalsIgnoreCase("START:")) {
				startSeen = true;
				continue;
			}

			if (str.equalsIgnoreCase(":END")) {
				endSeen = true;
				continue;
			}

			if (!startSeen) {
				continue;
			}

			if (!Character.isLetterOrDigit(str.charAt(0))) {
				continue;
			}

			if (str.equalsIgnoreCase("data:")) {
				mode = CurrentMode.DATA;

			} else if (str.equalsIgnoreCase("data-backup:")) {
				System.out.println();
				mode = CurrentMode.DATA_BACKUP;

			} else if (str.equalsIgnoreCase("password-backup:")) {
				System.out.println();
				mode = CurrentMode.PASSWORD_BACKUP;
			} else {

				if (mode == CurrentMode.DATA) {
//					String[] line = extractTag(str);
					ExtractTagReturn line = extractTag(str);

					assertValidDataAnnotations(line.annotations);

					boolean encrypted = line.annotations.contains(Data.ANNOTATION_ENCRYPTED);
					boolean noCloudNeeded = line.annotations.contains(Data.ANNOTATION_NO_CLOUD_NEEDED);
					boolean noLocalNeeded = line.annotations.contains(Data.ANNOTATION_NO_LOCAL_NEEDED);
					boolean oneBackupOnly = line.annotations.contains(Data.ANNOTATION_ONE_BACKUP_ONLY);

					System.out.println("Adding data: " + line.text);
					context.putData(
							new Data(line.text, line.tag, encrypted, noCloudNeeded, noLocalNeeded, oneBackupOnly));

				} else if ((mode == CurrentMode.DATA_BACKUP || mode == CurrentMode.PASSWORD_BACKUP)
						&& str.contains("->")) {
					String[] pair = getStringPair(str);

					if (mode == CurrentMode.DATA_BACKUP) {
						ExtractTagReturn line = extractTag(pair[1]);
						String text = line.text;
						String tag = line.tag;

						assertValidDataBackupAnnotations(line.annotations);

						System.out.println("Adding data backup: " + pair[0] + " -> " + text);
						context.addDataBackup(pair[0], text, tag, line.annotations);

					} else if (mode == CurrentMode.PASSWORD_BACKUP) {
						System.out.println("Adding password backup: " + pair[0] + " -> " + pair[1]);

						context.addPasswordBackup(pair[0], pair[1]);

//						IPasswordBackupable pb = context.getEntity(pair[0]);
//						pb.addPasswordBackup(pair[1]);

					} else {
						System.err.println("Ignoring: " + str);
					}

				} else {
					System.err.println("Ignoring line: " + str);
				}
			}

		}

		return context;

	}

	private static void assertValidDataAnnotations(List<String> annotations) {
		for (String str : annotations) {

			// Skip empty
			if (str.trim().length() == 0) {
				continue;
			}

			boolean match = false;
			for (String constAnnot : Data.ANNOTATIONS) {
				if (constAnnot.equalsIgnoreCase(str)) {
					match = true;
					break;
				}
			}

			if (!match) {
				throw new RuntimeException("Invalid annotation found: " + annotations);
			}
		}

	}

	private static void assertValidDataBackupAnnotations(List<String> annotations) {
		for (String str : annotations) {

			// Skip empty
			if (str.trim().length() == 0) {
				continue;
			}

			boolean match = false;
			for (String constAnnot : DataBackup.ANNOTATIONS) {
				if (constAnnot.equalsIgnoreCase(str)) {
					match = true;
					break;
				}
			}

			if (!match) {
				throw new RuntimeException("Invalid annotation found: " + annotations);
			}
		}

	}

//	private static String[] extractTagOld(String str) {
//		int openPos = str.indexOf("[");
//		int closePos = str.indexOf("]");
//		
//		if(openPos == -1 && closePos == -1) {
//			return new String[] {str, null };
//		}
//		
//		if(openPos == -1 || closePos == -1 ) {
//			throw new RuntimeException("Invalid format on line: "+str);
//		}
//		
//		String thing = str.substring(0, openPos).trim().toLowerCase();
//		
//		String tag = str.substring(openPos+1, closePos).trim().toLowerCase();
//		
//		return new String[] {thing, tag};		
//		
//	}

	private static ExtractTagReturn extractTag(String str) {
		int openPos = str.indexOf("[");
		int closePos = str.indexOf("]");

		if (openPos == -1 && closePos == -1) {
			throw new RuntimeException("No tag listed for: " + str);
		}

		if (openPos == -1 || closePos == -1) {
			throw new RuntimeException("Invalid format on line: " + str);
		}

		String text = str.substring(0, openPos).trim().toLowerCase();

		String tag = str.substring(openPos + 1, closePos).trim().toLowerCase();

		String annotations = str.substring(closePos + 1);

		List<String> annotResult = new ArrayList<>();
		String[] annots = annotations.split(" ");
		for (int x = 0; x < annots.length; x++) {
			String q = annots[x].trim();
			if (q.length() == 0) {
				continue;
			}

			if (!q.startsWith("@")) {
				throw new RuntimeException("Invalid annotation format on line: " + str);
			} else {
				annotResult.add(q.substring(1).trim().toLowerCase());
			}
		}

		ExtractTagReturn result = new ExtractTagReturn();
		result.annotations = annotResult;
		result.tag = tag;
		result.text = text;

		return result;

	}

	private static String[] getStringPair(String str) {
		int arrowPos = str.indexOf("->");
		String first = str.substring(0, arrowPos).trim();

		String second = str.substring(arrowPos + 2).trim();

		return new String[] { first, second };

	}

	private static class ExtractTagReturn {
		String text;
		String tag;
		List<String> annotations;
	}
}
