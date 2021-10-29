package com.jgw.internal;

import java.util.ArrayList;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class Data implements IPasswordBackupable {

	public static final String ANNOTATION_ENCRYPTED = "encrypted".toLowerCase();
	public static final String ANNOTATION_NO_CLOUD_NEEDED = "nocloudneeded".toLowerCase();
	public static final String ANNOTATION_ONE_BACKUP_ONLY = "onebackuponly".toLowerCase();
	public static final String ANNOTATION_NO_LOCAL_NEEDED = "nolocalneeded".toLowerCase();

	public static final String[] ANNOTATIONS = { ANNOTATION_ENCRYPTED, ANNOTATION_NO_CLOUD_NEEDED,
			ANNOTATION_NO_LOCAL_NEEDED, ANNOTATION_ONE_BACKUP_ONLY };

	private final String name;

	private final String tag;

	private final Map<String /* backup name */, DataBackup> dataBackups = new HashMap<>();

	private final List<Data> passwordBackups = new ArrayList<>();

	private final boolean encrypted;

	private final boolean noCloudNeeded;

	private final boolean oneBackupOnly;

	private final boolean noLocalNeeded;

	public Data(String name, String tag, boolean encrypted, boolean noCloudNeeded, boolean noLocalNeeded,
			boolean oneBackupOnly) {
		this.name = name;
		this.tag = tag;
		this.encrypted = encrypted;
		this.noCloudNeeded = noCloudNeeded;
		this.oneBackupOnly = oneBackupOnly;
		this.noLocalNeeded = noLocalNeeded;
	}

	public String getName() {
		return name;
	}

	protected void addPasswordBackup(Data passwordBackup) {
		passwordBackups.add(passwordBackup);
	}

	public List<Data> getPasswordBackups() {
		return Collections.unmodifiableList(passwordBackups);
	}

	protected void addDataBackup(DataBackup backup) {
		DataBackup test = dataBackups.get(backup.getName().toLowerCase());
		if (test != null) {
			throw new RuntimeException("Duplicate data backup: " + backup.getName());
		}

		dataBackups.put(backup.getName().toLowerCase(), backup);

	}

	public List<DataBackup> getDataBackups() {
		List<DataBackup> result = new ArrayList<DataBackup>();
		result.addAll(dataBackups.values());
		return result;
	}

	public Map<String, DataBackup> getDataBackupsMap() {
		return Collections.unmodifiableMap(dataBackups);
	}

	public String getTag() {
		return tag;
	}

	@Override
	public String toString() {
		return name + " [" + tag + "]";
	}

	public boolean isEncrypted() {
		return encrypted;
	}

	public boolean isNoCloudNeeded() {
		return noCloudNeeded;
	}

	public boolean isOneBackupOnly() {
		return oneBackupOnly;
	}

	public boolean isNoLocalNeeded() {
		return noLocalNeeded;
	}

}
