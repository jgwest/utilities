package com.jgw.internal;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;

public class DataBackup implements IPasswordBackupable {

	public final static String ANNOTATION_UNENCRYPTED = "unencrypted";

	public final static String[] ANNOTATIONS = new String[] { ANNOTATION_UNENCRYPTED };

	private final String name;

	private final List<Data> passwordBackups = new ArrayList<>();

	private final String tag;

	private boolean unencrypted;

	public DataBackup(String name, String tag, boolean unencrypted) {
		this.name = name;
		this.tag = tag;
		this.unencrypted = unencrypted;
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

	public String getTag() {
		return tag;
	}

	@Override
	public String toString() {
		return name + " [" + tag + "]";
	}

	public boolean isUnencrypted() {
		return unencrypted;
	}
}
