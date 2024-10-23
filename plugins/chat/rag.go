/*
Copyright © 2021 eryajf

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package chat

import (
	"fmt"

	log "github.com/cihub/seelog"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "langchaingo-ollama-rag",
	Short: "学习基于langchaingo构建的rag应用",
	Long:  `学习基于langchaingo构建的rag应用`,
}


func init() {
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	// ========
	rootCmd.AddCommand(FileToChunksCmd)
	FileToChunksCmd.Flags().StringP("filepath", "f", "test.txt", "指定文件路径, 默认为test.txt")
	FileToChunksCmd.Flags().IntP("chunksize", "c", 200, "指定块大小，默认为100")
	FileToChunksCmd.Flags().IntP("chunkoverlap", "o", 50, "指定块重叠大小，默认为10")
	// ========
	rootCmd.AddCommand(EmbeddingCmd)
	EmbeddingCmd.Flags().StringP("filepath", "f", "test.txt", "指定文件路径, 默认为test.txt")
	EmbeddingCmd.Flags().IntP("chunksize", "c", 200, "指定块大小，默认为100")
	EmbeddingCmd.Flags().IntP("chunkoverlap", "o", 50, "指定块重叠大小，默认为10")
	// ========
	rootCmd.AddCommand(RetrieverCmd)
	RetrieverCmd.Flags().IntP("topk", "t", 5, "召回数据的数量，默认为5")

}

var FileToChunksCmd = &cobra.Command{
	Use:   "filetochunks",
	Short: "将文件转换为块儿",
	Run: func(cmd *cobra.Command, args []string) {
		filepath, _ := cmd.Flags().GetString("filepath")
		chunkSize, _ := cmd.Flags().GetInt("chunksize")
		chunkOverlap, _ := cmd.Flags().GetInt("chunkoverlap")

		docs, err := TextToChunks(filepath, chunkSize, chunkOverlap)
		if err != nil {
			log.Error("转换文件为块儿失败，错误信息: %v", err)
		}
		log.Info("转换文件为块儿成功，块儿数量: ", len(docs))
		for _, v := range docs {
			fmt.Printf("🗂 块儿内容==> %v\n", v.PageContent)
		}
	},
}

var EmbeddingCmd = &cobra.Command{
	Use:   "embedding",
	Short: "将文档块儿转换为向量",
	Run: func(cmd *cobra.Command, args []string) {
		filepath, _ := cmd.Flags().GetString("filepath")
		chunkSize, _ := cmd.Flags().GetInt("chunksize")
		chunkOverlap, _ := cmd.Flags().GetInt("chunkoverlap")
		docs, err := TextToChunks(filepath, chunkSize, chunkOverlap)
		if err != nil {
			log.Error("转换文件为块儿失败，错误信息: %v", err)
		}
		err = storeDocs(docs, getStore())
		if err != nil {
			log.Error("转换块儿为向量失败，错误信息: %v", err)
		} else {
			log.Info("转换块儿为向量成功")
		}
	},
}

var RetrieverCmd = &cobra.Command{
	Use:   "retriever",
	Short: "将用户问题转换为向量并检索文档",
	Run: func(cmd *cobra.Command, args []string) {
		topk, _ := cmd.Flags().GetInt("topk")

		// 获取用户输入的问题
		prompt, err := GetUserInput("请输入你的问题")
		if err != nil {
			log.Error("获取用户输入失败，错误信息: %v", err)
		}
		rst, err := useRetriaver(getStore(), prompt, topk)
		if err != nil {
			log.Error("检索文档失败，错误信息: %v", err)
		}
		for _, v := range rst {
			fmt.Printf("🗂 根据输入的内容检索出的块儿内容==> %v\n", v.PageContent)
		}
	},
}

