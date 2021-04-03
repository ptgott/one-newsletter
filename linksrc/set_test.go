package linksrc

// Using one HTML string in all unit tests. Just like
// in a real case, we can't change the HTML we want to
// scrape.
const testHTML = `<!doctype html5>
<html>
<head>
</head>
<body>
	<h1>This is my cool website</h1>
	<div id="mostRead">
		<h2>Most read posts today</h2>
		<ul>
			<li>
				<img src="img1.png">A cool image</img>
				<span class="itemHolder">
					<span class="itemNumber">1.</span>
					<span class="itemName">This is a hot take!</span>
				</span>
				<a href="www.example.com/stories/hot-take">
				Click here
				</a>
			</li>
			<li>
				<img src="img2.png">This is an image</img>
				<span class="itemHolder">
					<span class="itemNumber">2.</span>
					<span class="itemName">Stuff happened today, yikes.</span>
				</span>
				<a href="www.example.com/stories/stuff-happened">
				Click here
				</a>
			</li>
			<li>
				<img src="img3.png">This is also an image</img>
				<span class="itemHolder">
					<span class="itemNumber">3.</span>
					<span class="itemName">Is this supposition really true?</span>
				</span>
				<a href="www.example.com/storiesreally-true">
				Click here
				</a>
			</li>
		<ul>
	</div>
</body>
</html>`
