import subprocess
import yaml
import tempfile
import os

def get_istio_config():
    try:
        # Run kubectl command and capture the output
        command = "kubectl get configmap istio -n istio-system -o yaml"
        result = subprocess.run(command, shell=True, capture_output=True, text=True)
        
        # Check if the command was successful
        if result.returncode == 0:
            # Parse the YAML content
            yaml_data = yaml.safe_load(result.stdout)
            
            # Extract the desired part
            config_part = yaml_data.get('data', {}).get('mesh', '')
            
            # If you want to further process it as a Python dictionary, you can use yaml.safe_load
            config_dict = yaml.safe_load(config_part)
            return yaml_data, config_dict
        else:
            print(f"Error running kubectl command: {result.stderr}")
    except Exception as e:
        print(f"An error occurred: {e}")

def add_access_logging(config):
    # Check if 'defaultProviders' key exists
    if 'defaultProviders' in config:
        # Check if 'accessLogging' key exists within 'defaultProviders'
        if 'accessLogging' not in config['defaultProviders']:
            # If it doesn't exist, create it and set its value
            config['defaultProviders']['accessLogging'] = ['boanlab-collector-1']
            print("Added boanlab-collector-1 to defaultProviders.accessLogging")
    else:
        # If 'defaultProviders' key doesn't exist, create it along with 'accessLogging'
        config['defaultProviders'].append({'accessLogging': ['boanlab-collector-1']})
        print("Added boanlab-collector-1 to defaultProviders.accessLogging")

    return config

def add_extension_providers(config):
    # Check if 'extensionProviders' key exists
    if 'extensionProviders' not in config:
        # If it doesn't exist, create it and set its value
        config['extensionProviders'] = [
            {
                'name': 'boanlab-collector-1',
                'envoyOtelAls': {
                    'service': 'custom-collector.collector-1.svc.cluster.local',
                    'port': 4317
                }
            }
        ]
        print("Added boanlab-collector-1 to extensionProviders")
    else:
        config["extensionProviders"].append({
            'name': 'boanlab-collector-1',
            'envoyOtelAls': {
                'service': 'custom-collector.collector-1.svc.cluster.local',
                'port': 4317
            }
        })
        print("Added boanlab-collector-1 to extensionProviders")

    return config


def edit_configmap(config):
    try:
        # Convert the modified 'mesh' part to YAML
        modified_yaml_content = yaml.dump(config, default_flow_style=False)

        # Create a temporary file to store the modified YAML content
        temp_file_path = tempfile.mktemp(suffix=".yaml", prefix="istio_config_")
        temp_file_path="test.yaml"
        with open(temp_file_path, "w") as temp_file:
            content = modified_yaml_content.replace("mesh:", "mesh: |-")
            temp_file.write(content)

        # Run kubectl patch to edit the ConfigMap using the temporary file
        command = f"kubectl patch configmap istio -n istio-system --type merge --patch-file {temp_file_path}"
        subprocess.run(command, shell=True, check=True)
        print("ConfigMap edited successfully.")

    except Exception as e:
        print(f"An error occurred while editing the ConfigMap: {e}")
    finally:
        # Cleanup: Delete the temporary file
        if temp_file_path and os.path.exists(temp_file_path):
            os.remove(temp_file_path)

# Setup collectors and exporters for our system
try:
    command = f"kubectl create -f ./k8s"
    subprocess.run(command, shell=True, check=True)
except Exception as e:
    print(f"An error occurred while applying resources: {e}")

# Call the function to get the Istio configuration
raw, istio_config = get_istio_config()

# Add access logging if not present
istio_config = add_access_logging(istio_config)
istio_config = add_extension_providers(istio_config)

# Update the mesh part
raw['data']['mesh'] = istio_config

# Convert the modified 'mesh' part to YAML
modified_yaml_content = yaml.dump(raw, default_flow_style=False)

# Apply the modified YAML content to the ConfigMap
edit_configmap(raw)

print("Injecting sidecars...")
try:
    command = f"bash k8s/inject-sidecars.sh"
    subprocess.run(command, shell=True, check=True)
except Exception as e:
    print(f"An error occurred while injecting side cars: {e}")
