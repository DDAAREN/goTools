package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	log "github.com/golang/glog"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"
)

var (
	deployment string
	kubeMaster string
	kubeConfig string = `apiVersion: v1              
clusters:                   
- cluster:                  
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUN5RENDQWJDZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFWTVJNd0VRWURWUVFERXdwcmRXSmwKY201bGRHVnpNQjRYRFRFNE1EY3dOVEEzTkRrd01Gb1hEVEk0TURjd01qQTNORGt3TUZvd0ZURVRNQkVHQTFVRQpBeE1LYTNWaVpYSnVaWFJsY3pDQ0FTSXdEUVlKS29aSWh2Y05BUUVCQlFBRGdnRVBBRENDQVFvQ2dnRUJBTXEyCmJ0ckhkQWZYaWJmdjB2Tkh2Q2VqOHQ1dkZ0U0ZsSUgvczdLZlVQY1prUE9KcHR1cENNb05ZamdWU0NROUx2cTIKd3lUN1h6QkFwUkR1TUZCMXV3ejYzbGx6YjRhdnYyV25JUCtkK25TbUE3enlRdXlKUmlqV0REZldZRzZ2cFFhVQpoUUNXbVZpRU9YdFhGb1ZHQnlTejE3S1VYek1JOTVzS2pXdkNJWXVEN2pheWl0WDhIYjMrekJMRUFoQmMwMzJpCmNBemxaUjVIdVpFanRxN2xRWWNpbnlBZzdlVG1ic0ZDT01aeHhIYjA0akxsa0gvMDdHY0hmcFBYdlloTFJpRmYKSGdNblljL2xHaGRia2prUTczNXg2SXZibXNiUFcrdVlGNncvQ0VLSEtpZGgxREI0VnQ5R210eTR4T3h5a2Q4TAorVlh1WnVVSFkyTExtbjB0ZlNFQ0F3RUFBYU1qTUNFd0RnWURWUjBQQVFIL0JBUURBZ0trTUE4R0ExVWRFd0VCCi93UUZNQU1CQWY4d0RRWUpLb1pJaHZjTkFRRUxCUUFEZ2dFQkFCUmMzRUdRdGlLc2dFZ0Z2WWZ1aVh5MVdMWTgKRmJNaWtUazFaTGtYbUdXOHUxNEY3N21OTUk1MFZrQnZCOFhUQ1NmS1NOVzVPZWRjdndlMzdxS3hoZ2pkZ0l2Ywp3cmVWSWlCNTNnbTE0aGxjU0Z2cXRtUWVPNVM3dTg0c3hHVnJoalhKaUx0OTF4TkN2cXlOQlZmRUd4VHJBUHRkCmE2L2ZoZzduOFY0S3cyMFRMUjlhV20wYWJiZDN0ODUrTkorVDFjNFNTcThhaWVoYmhiN2tDd0c0MTBGNi9WTDUKM3ozVHdNZjNRR1llemJ6UkE2Q2tDY0JFdkpreEc0L2ZscTQ2aXp5N3JOQmJjOEdtRm10Qk96ZjNqbHlLQlZMVwpidm9MRi9vVHJNZVg3SGNVdGhoa3luTElDM0p1L0Joci85dEhQNkkydVRrVWZPQnh4YmthYWhvR2JNTT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
    server: https://10.252.16.23:6443                    
  name: kubernetes          
contexts:                   
- context:                  
    cluster: kubernetes     
    user: kubernetes-admin  
  name: kubernetes-admin@kubernetes                      
current-context: kubernetes-admin@kubernetes             
kind: Config                
preferences: {}             
users:                      
- name: kubernetes-admin    
  user:                     
    client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUM4akNDQWRxZ0F3SUJBZ0lJR2tWbU04L1NreWN3RFFZSktvWklodmNOQVFFTEJRQXdGVEVUTUJFR0ExVUUKQXhNS2EzVmlaWEp1WlhSbGN6QWVGdzB4T0RBM01EVXdOelE1TURCYUZ3MHhPVEEzTURVd056UTVNRFJhTURReApGekFWQmdOVkJBb1REbk41YzNSbGJUcHRZWE4wWlhKek1Sa3dGd1lEVlFRREV4QnJkV0psY201bGRHVnpMV0ZrCmJXbHVNSUlCSWpBTkJna3Foa2lHOXcwQkFRRUZBQU9DQVE4QU1JSUJDZ0tDQVFFQXlCSWJxWmdzV0JOak9icVYKMmNPbXBFSnAwRlZHcVJYNlNMVi8rZlV4Wis1ZnpweG1IYUlhRHZQSTV4dEFkMHZPUHQ3Y1FKUitOZ0svdTFSWgpCeEhXbWlXTExKc3Z1K1k0T3BIdXBxSEhLZ2FTMUVscFowOTNoWnR3ZzBsV1ZOUFplakRCSTdqTitrZ3pOS3FVCjQ1cThQQlVES3BGd0FLLzJvZEJwM3BwenZlK2FBOVhDYTUyS3lxSmJSYlRDS1ZGMzBmemxnQ2ZoRVh5YUxqRGgKdllBU0VTY2EvTWhUSTBhbW5IeTJ6QmtVSXZHanJYbFMxTmxxUUpUenExWDhxVHZ0MFB4N0hsd2h6UWt3OGJZMApmakMzcjhmWU1JZVgzWUFwa2Nabkg1UXhIY0RXU1FESjZOWG8vWW9qV2ZYWWk3MzB0WSswdHQ4ZW1HMDNJc1VRCmtyTzRyUUlEQVFBQm95Y3dKVEFPQmdOVkhROEJBZjhFQkFNQ0JhQXdFd1lEVlIwbEJBd3dDZ1lJS3dZQkJRVUgKQXdJd0RRWUpLb1pJaHZjTkFRRUxCUUFEZ2dFQkFJdzV2V2F6QjJlSXhvVmJlcFZxV09pYUJyY3kxU3BMQVJwdApIc21vaVhhNG9WWERRcEkzdTF3TWtNZ1JxSHNGUEdNVitQL01GWEpuRjZCUTBQVVR2dkhVVG1zSHh4bTB0RDhGCk9LdXF2M0hBMzBDZGdwZjNPSDc1ZU1pOHlkMUF3a2d1YXpzSnNsNCtBa2s2TDlIMnM3ZjVrVURSK2JPZEhFWUQKdXNnMW9yblZWajNlaUZ6M2gzaXNMVlV4TTdHM0pRNHQxK3JNb21wU1JIM0svVXhIRVNUZlgyTi9aN3F5UzRuUApLZ3ZQbDdKakdNN0tMNWhUMTJDdXZJTlRpVHIvcFFwNTBSVFNmSWxxbEpDMHRvaDRwQ2IvbzloelR5MlZNMEVDCkdiazdyeEhuVUxlUjc2MnFCaldSR3FISVR2RDJOTWErdlV3YTNvaE5ZWVUvQ3RObWxzQT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
    client-key-data: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFcFFJQkFBS0NBUUVBeUJJYnFaZ3NXQk5qT2JxVjJjT21wRUpwMEZWR3FSWDZTTFYvK2ZVeForNWZ6cHhtCkhhSWFEdlBJNXh0QWQwdk9QdDdjUUpSK05nSy91MVJaQnhIV21pV0xMSnN2dStZNE9wSHVwcUhIS2dhUzFFbHAKWjA5M2hadHdnMGxXVk5QWmVqREJJN2pOK2tnek5LcVU0NXE4UEJVREtwRndBSy8yb2RCcDNwcHp2ZSthQTlYQwphNTJLeXFKYlJiVENLVkYzMGZ6bGdDZmhFWHlhTGpEaHZZQVNFU2NhL01oVEkwYW1uSHkyekJrVUl2R2pyWGxTCjFObHFRSlR6cTFYOHFUdnQwUHg3SGx3aHpRa3c4YlkwZmpDM3I4ZllNSWVYM1lBcGtjWm5INVF4SGNEV1NRREoKNk5Yby9Zb2pXZlhZaTczMHRZKzB0dDhlbUcwM0lzVVFrck80clFJREFRQUJBb0lCQVFDWGVyNWZCY3h0VXlDSgprTy9zVk9wUzY4WFo2dmI1QnA3ZGRpNVRQb1lOdnJuallSOGZ5S0FhT2hJZWlNK2lnMDdBNDFPM3diWmRobTlNCmtteGxvZWZ1QmdiOTJ2R2xQb1hNTXJtU2lHS1hPeXJvZUR6Sjc2ejdCOW1FVFg2RDgzSnh4WUEyWDdIMGtiM1QKWG1Ra2YvK05MZ3VicTBvMSt3U3ROM014QzVnZWNYaGZETkQxQ2JwOXhHZ1pGRVBxMldTM3VPREdRZVVDSjYxcQpMSCtadnROdUxyNE1hUWI4V3B0UVRCbldMSnBlOTk0RWJWa1RwRUY0Vjc4MGhpV0kzc1RiVmozRUhTZ0hkaXpBCkhQeGYvT08rczF2SFNkYXdFd1U2WVZLVlZTakttbUR0Z01hWVZrNkN0SDVMc2Y2QVQ2Y2dSWGxqVE9RNkhZN3kKOGIrelVvUUJBb0dCQU9BM0hWOCtuZ3cvN2NNNFNLYXdKOTNOVWpNTVJxc0ZXMUgzcVEvdnQxNlNBSGlGTEIzRQpjM1A5QkgwbHoyTmxFbVJ1SURqek44cnpBd3IxeDJwY0JJY1Mxb0F5ajkxNVh1cHNFcHllMXdiYVVvWTFBcHc4CjR1bmwwVllObVVjRHg4NGoyNUJ0MmZEc1ZQaWNTSFhyWHVwa0E0d29XNUZDRXNKcUd5WUFaNUh0QW9HQkFPUnUKeG5ZeGdPNVZqUFlyWFU0NVRpd2ZEWllVMnB4eWR1MWNMdE1IeldFeEYxZHZvT1ZxaWRzcmJvUE9meGlCaExWSgo1WkY4eGJDU3JTd3d0ZXBTcEcvL2NxNTluNVpnTGlvWDRpUVlRcDdDRUhTSkdia2E0UE9Ua25BeFB3dmRVMHkrCkdNejNWbEt5OFpqR2pMU3Y4T0krQUZYVWRnUkxEa2VsM0NkQ1J1bkJBb0dCQU5GRHYzODRveXhOc216RktGR2oKRWVKYkVzQWdVZ2ltbkQvWmhZb2hNeVRwNGRTYWZyMWRzRi91STNWbWg3UitEZmQ4TFVqYUFCWEVUKyszeXlKQwp0ZHNYd3VtdHgwWnZWQjQ1TmZuRjZtMHo4VmZmUEF0MGJGamZyVXpDcm05d1lOak44TXhSS3R0SXlGbXRDNWc3ClVNQTFEbmFPNkQrZnlvQjNwZFVIQmFOVkFvR0FYaUYveXFpdmxvYk9aWXFORW5UdXo2T2tONW8wVTQrZmprUVUKVDRYQmpqRnFpdTlIQUFLYytDRzNrcnovQnB3b2tZUDRBN0hFelBSRVJCZDJmeTY2OENQMW9BM0lPM0U2MU1HdQp3R3oyMXZEbFV3QkVCMUVhTFlVOExOcytQYWRnY2hsTG92cXhLYmJ2YzZNdHpDOU1OZzZTbU12S0xnNjN2YktOCk8raEZ6SUVDZ1lFQW96QzVORDJJUWdDSGtKd05zQTdJRnZaUzllK1RsdmdiU1Vzd210RjNRN0VxMCtYYndacWUKMXVYcU1XV2lLMmczK0NzUS9KZkhXb2JxdmxnOFRjYVRTMnFEL3NZSXNRK0VBTXNKVUVJTTJVV0p4YjkvdW5hSgorUEFWamtVQXBRYlpCUURsNlUxc00vbjNaK1p6RFIzQ1VCZWVkQk5sL1NYMXVIR3hNclE5dXNRPQotLS0tLUVORCBSU0EgUFJJVkFURSBLRVktLS0tLQo=`
)

const (
	NAMESPACE string = "machine-learning"
)

type K8sCluster struct {
	ApiServerUrl string
}

func init() {
	log.V(5)
	log.Flush()
	//flag.StringVar(&kubeConfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	flag.StringVar(&kubeMaster, "master", "https://10.252.16.23:6443", "master url")
	flag.StringVar(&deployment, "deployment", "", "deployment name")
	flag.Parse()
}

func main() {
	//config, err := clientcmd.BuildConfigFromFlags(kubeMaster, kubeConfig)
	err := ioutil.WriteFile("/tmp/.kubeConfig", []byte(kubeConfig), 0644)
	if err != nil {
		log.Fatal(err)
	}
	config, err := clientcmd.BuildConfigFromFlags(kubeMaster, "/tmp/.kubeConfig")
	if err != nil {
		log.Fatal(err)
	}
	os.Remove("/tmp/.kubeConfig")

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	deploymentsClient := clientset.AppsV1().Deployments(NAMESPACE)
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Retrieve the latest version of Deployment before attempting update
		// RetryOnConflict uses exponential backoff to avoid exhausting the apiserver
		result, getErr := deploymentsClient.Get(deployment, metav1.GetOptions{})
		if getErr != nil {
			panic(fmt.Errorf("Failed to get latest version of Deployment: %v", getErr))
		}
		nowStr := time.Now().Format("2006-01-02T15:04:05Z07:00")
		found := false
		for i, e := range result.Spec.Template.Spec.Containers[0].Env {
			if e.Name == "restartAt" {
				result.Spec.Template.Spec.Containers[0].Env[i].Value = nowStr
				found = true
				break
			}
		}
		if !found {
			result.Spec.Template.Spec.Containers[0].Env = append(result.Spec.Template.Spec.Containers[0].Env, apiv1.EnvVar{Name: "restartAt", Value: nowStr})
		}
		_, updateErr := deploymentsClient.Update(result)
		//		fmt.Printf("%#v", result)
		return updateErr
	})
	if retryErr != nil {
		panic(fmt.Errorf("Update failed: %v", retryErr))
	}
}
