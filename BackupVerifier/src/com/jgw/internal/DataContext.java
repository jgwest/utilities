package com.jgw.internal;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class DataContext {

	private final Map<String /* data name */, Data> dataMap = new HashMap<>();

	private final Map<String /* data backup name */, DataBackup> dataBackupMap = new HashMap<>();

	public DataContext() {
	}

	public void putData(Data d) {
		Data exists = dataMap.get(d.getName().toLowerCase());
		if (exists != null) {
			throw new RuntimeException("Data already exists.");
		}

		dataMap.put(d.getName().toLowerCase(), d);
	}

//	private void putDataBackup(DataBackup d) {
//		DataBackup exists = dataBackupMap.get(d.getName().toLowerCase());
//		if(exists != null) {
//			throw new RuntimeException("Data backup already exists.");
//		}
//		
//		
//		dataBackupMap.put(d.getName().toLowerCase(), d);
//	}

	public Data getData(String dataName) {
		Data result = dataMap.get(dataName.toLowerCase());
		if (result == null) {
			throw new RuntimeException("Data not found: " + dataName);
		}

		return result;
	}

//	private DataBackup getDataBackup(String dataBackupName) {
//		DataBackup result = dataBackupMap.get(dataBackupName.toLowerCase());
//		if(result == null)  {
//			throw new RuntimeException("Data Backup not found: "+dataBackupName);
//		}
//		
//		return result;		
//		
//	}

	public IPasswordBackupable getEntity(String name) {
		Data data = dataMap.get(name.toLowerCase());
		DataBackup databackup = dataBackupMap.get(name.toLowerCase());

		if (data == null && databackup == null) {
			throw new RuntimeException("Could not find entity: " + name);
		}

		if (data != null && databackup != null) {
			throw new RuntimeException("two entities with the same name: " + name);
		}

		return data != null ? data : databackup;

	}

	public void addDataBackup(String dataName, String dataBackupName, String tag, List<String> annotations) {
		Data d = getData(dataName.toLowerCase());

		// Check: dataBackupName cannot have same name as existing data
		if (dataMap.get(dataBackupName.toLowerCase()) != null) {
			throw new RuntimeException("Data specified as a data backup: " + dataName + " -> " + dataBackupName);
		}

		boolean unencrypted = annotations.contains(DataBackup.ANNOTATION_UNENCRYPTED);

		DataBackup db = dataBackupMap.get(dataBackupName.toLowerCase());
		if (db == null) {
			db = new DataBackup(dataBackupName.toLowerCase(), tag, unencrypted);
			dataBackupMap.put(dataBackupName.toLowerCase(), db);
		} else {
			// Check: Unencrypted annotation must match what was specified for any previous
			// dataBackups with this name
			if (unencrypted != db.isUnencrypted()) {
				throw new RuntimeException("Mismatch on Unencrypted annotation: " + dataName + " -> " + dataBackupName);
			}
		}

		d.addDataBackup(db);
	}

	public void addPasswordBackup(String entityName, String passwordDataBackupParam) {

		IPasswordBackupable entity = getEntity(entityName);

		for (Data d : entity.getPasswordBackups()) {
			// look to see if the entity already has this password backup
			if (d.getName().toLowerCase().equalsIgnoreCase(passwordDataBackupParam.toLowerCase())) {
				throw new RuntimeException(
						"Duplicate Password Data Backup: " + entityName + " -> " + passwordDataBackupParam);
			}
		}

		Data passwordDataBackup = getData(passwordDataBackupParam);

		if (entity instanceof Data) {
			((Data) entity).addPasswordBackup(passwordDataBackup);

		} else if (entity instanceof DataBackup) {
			((DataBackup) entity).addPasswordBackup(passwordDataBackup);

		} else {
			throw new RuntimeException();
		}

	}

	public List<Data> getAllData() {
		List<Data> result = new ArrayList<>();
		result.addAll(dataMap.values());
		return result;
	}

}
