package structuredoutput

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

// ============ 输入结构 ============

type MovieReviewInput struct {
	ReviewText string `json:"review_text"` // 用户的评论文本
}

// ============ 输出结构（包含所有 JSON Schema 约束示例）============

type MovieReviewAnalysis struct {
	MovieTitle     string           `json:"movie_title" jsonschema:"description=电影名称,required,minLength=1,maxLength=100"`
	Genre          string           `json:"genre" jsonschema:"description=电影类型,required,enum=action,enum=comedy,enum=drama,enum=horror,enum=romance,enum=scifi,enum=other"`
	ReleaseYear    string           `json:"release_year" jsonschema:"description=上映年份(YYYY格式),required,pattern=^(19|20)[0-9]{2}$"`
	ReviewDate     string           `json:"review_date" jsonschema:"description=评论日期,required,format=date"`
	Language       string           `json:"language" jsonschema:"description=评论语言,default=zh-CN,enum=zh-CN,enum=en-US,enum=ja-JP"`
	Rating         int              `json:"rating" jsonschema:"description=评分(1-10分),required,minimum=1,maximum=10"`
	PriceWorth     float64          `json:"price_worth" jsonschema:"description=性价比评分(0-5分_0.5递增),required,minimum=0,maximum=5,multipleOf=0.5"`
	Confidence     float64          `json:"confidence" jsonschema:"description=分析置信度(0-1之间_不含边界),required,exclusiveMinimum=0,exclusiveMaximum=1"`
	RecommendLevel int              `json:"recommend_level" jsonschema:"description=推荐等级,required,enum=1,enum=2,enum=3,enum=4,enum=5"`
	IsSpoiler      bool             `json:"is_spoiler" jsonschema:"description=是否包含剧透,required"`
	IsRecommended  bool             `json:"is_recommended" jsonschema:"description=是否推荐观看,default=true"`
	Tags           []string         `json:"tags" jsonschema:"description=标签(至少1个_最多5个),required,minItems=1,maxItems=5"`
	Actors         []string         `json:"actors" jsonschema:"description=提到的演员(不重复),uniqueItems=true"`
	Highlights     []Highlight      `json:"highlights" jsonschema:"description=精彩片段(至少1个),required,minItems=1"`
	Sentiment      SentimentScore   `json:"sentiment" jsonschema:"description=情感分析,required"`
	Comparison     *MovieComparison `json:"comparison,omitempty" jsonschema:"description=与其他电影的比较(可选)"`
	ReviewerName   string           `json:"reviewer_name" jsonschema:"description=评论者昵称,examples=影迷小王,examples=电影爱好者"`
	Summary        string           `json:"summary" jsonschema:"title=评论摘要,description=一句话总结,required,minLength=10,maxLength=200"`
}

type Highlight struct {
	Timestamp   string `json:"timestamp" jsonschema:"description=时间戳(MM:SS格式),required,pattern=^[0-5][0-9]:[0-5][0-9]$"`
	Description string `json:"description" jsonschema:"description=片段描述,required,minLength=5,maxLength=100"`
	Type        string `json:"type" jsonschema:"description=片段类型,required,enum=action,enum=funny,enum=touching,enum=suspense"`
}

type SentimentScore struct {
	Overall       string  `json:"overall" jsonschema:"description=总体情感,required,enum=positive,enum=neutral,enum=negative"`
	PositiveScore float64 `json:"positive_score" jsonschema:"description=积极度(0-1),required,minimum=0,maximum=1"`
	NegativeScore float64 `json:"negative_score" jsonschema:"description=消极度(0-1),required,minimum=0,maximum=1"`
}

type MovieComparison struct {
	ComparedMovie string   `json:"compared_movie" jsonschema:"description=对比的电影,required,minLength=1"`
	Similarity    int      `json:"similarity" jsonschema:"description=相似度百分比(0-100),required,minimum=0,maximum=100"`
	BetterAspects []string `json:"better_aspects" jsonschema:"description=更好的方面,maxItems=5"`
}

func buildMovieReviewPrompt(ctx context.Context, input MovieReviewInput) ([]*schema.Message, error) {
	systemPrompt := `你是一个专业的电影评论分析专家。分析用户的评论文本，提取关键信息并进行情感分析。

**注意事项：**
1. 评分范围：1-10分（整数）
2. 性价比评分：0-5分，以0.5为递增单位
3. 上映年份：1900-2099年，格式YYYY
4. 至少提取1个精彩片段
5. 标签数量：1-5个
6. 时间戳格式：MM:SS

通过调用 analyze_movie_review 工具返回分析结果。`

	userPrompt := fmt.Sprintf("请分析以下电影评论：\n\n%s", input.ReviewText)

	return []*schema.Message{
		{
			Role:    schema.System,
			Content: systemPrompt,
		},
		{
			Role:    schema.User,
			Content: userPrompt,
		},
	}, nil
}

func TestChain_Invoke(t *testing.T) {
	openaiApiKey := os.Getenv("OPENAI_API_KEY")
	if openaiApiKey == "" {
		t.Skip("跳过测试：未设置 OPENAI_API_KEY 环境变量")
	}
	openaiModel := os.Getenv("OPENAI_MODEL")
	if openaiModel == "" {
		openaiModel = "gpt-4o"
	}
	baseUrl := os.Getenv("OPENAI_BASE_URL")
	if baseUrl == "" {
		baseUrl = "https://api.openai.com/v1"
	}
	ctx := context.Background()
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  openaiApiKey,
		Model:   openaiModel,
		BaseURL: baseUrl,
	})
	if err != nil {
		t.Fatalf("创建 ChatModel 失败: %v", err)
	}
	chain, err := NewChain[MovieReviewInput, MovieReviewAnalysis](
		chatModel,
		buildMovieReviewPrompt,
		"analyze_movie_review",
		"分析电影评论，提取结构化信息",
	)
	if err != nil {
		t.Fatalf("创建 Chain 失败: %v", err)
	}
	testReviews := []string{
		// 案例 1：正面评价
		`《星际穿越》真是太震撼了！诺兰导演的杰作，2014年上映至今仍是我心中的科幻片第一名。
        评分必须给满分10分！特别是那个黑洞的视觉效果，在01:23:45那段简直美哭了。
        马修·麦康纳和安妮·海瑟薇的演技也很在线。
        票价虽然贵但绝对值得，性价比给5分！
        强烈推荐给所有科幻迷，这部片子让人深思时间和爱的意义。
        #硬科幻 #视觉盛宴 #必看神作`,

		// 案例 2：中性评价
		`昨天看了《流浪地球2》，2023年的国产科幻片。
        整体来说还不错，给7分吧。特效确实做得很用心，
        在00:45:30的太空电梯那段很震撼，在01:15:20的月球爆炸也很壮观。
        刘德��、吴京演技都在线。
        性价比中等，给3.5分。
        适合喜欢科幻的朋友，但剧情有点复杂。
        和《星际穿越》比，视觉效果相似度大概70%，但叙事还有提升空间。
        #国产科幻 #特效不错`,

		// 案例 3：负面评价（包含剧透）
		`《XX恐怖片》2023年上映，真的不推荐。
        只能给3分，剧情套路，吓人手法老套。
        唯一的看点是00:33:12那段追逐戏还算紧张。
        性价比很低，只给1分。
        ⚠️ 剧透警告：最后男主角死了，女主角其实是鬼。
        如果你喜欢恐怖片，还不如去看经典老片。
        #不推荐 #浪费时间`,
	}
	for idx, review := range testReviews {
		t.Logf("----- 测试案例 %d -----", idx+1)
		input := MovieReviewInput{
			ReviewText: review,
		}
		result, err := chain.Invoke(ctx, input)
		if err != nil {
			t.Errorf("Invoke 失败: %v", err)
			continue
		}
		resultJson, _ := json.MarshalIndent(result, "", "  ")
		t.Logf("分析结果:\n%s", string(resultJson))
	}

}
